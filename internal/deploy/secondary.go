package deploy

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/svcpaths"
	"github.com/PavelNeyman/netductor/internal/version"
)

// SecondaryOpts — provision RU secondary from operator workstation (Mac).
//
// Path (locked):
//  1. Mac → primary SSH (key): prepare-pack (IssueToken + mTLS + VPN bundle) — no primary→secondary SSH
//  2. Mac → secondary SSH (password once): install pubkey, binary, secrets, join, harden-last
//
// Agent then talks primary :8789 over mTLS. Mac private key never stored on primary or secondary.
type SecondaryOpts struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string // Mac private key path (pubkey = path+".pub")
	PrimaryKeyPassphrase string
	// PrimarySSHPort is the SSH port on primary. 0 means 52222.
	PrimarySSHPort  int
	SecondaryHost   string
	SecondaryUser   string
	SecondaryPass   string
	SecondarySSHKey string // resume: key already on secondary. Empty uses PrimaryKey.
	// SSHMode: "initial" (port 22 + password) or "resume" (port 52222 + key).
	// Empty: password-only → initial; key-only → resume; both → error.
	SSHMode        string
	SNI            string
	OperatorPubKey string
	// ForceReinstall: operator already verified host-key rotation — pass through to Go TOFU forget.
	ForceReinstall bool
	// Promote is rejected. Traffic test is a separate operator step; deploy must not promote.
	Promote bool
	// PeerID overrides the default peer-<ip>. Required after a removed peer reused the same IP.
	PeerID string
	// ViaPrimary is ignored (deprecated). Always Mac-direct.
	ViaPrimary bool
}

const (
	SSHModeInitial = "initial"
	SSHModeResume  = "resume"
)

// ResolveSecondarySSH picks provision port and auth from mode.
// initial: 22 + password. resume: 52222 + key (SecondarySSHKey or PrimaryKey).
// Empty mode infers from which credential is set. Both set without mode is an error.
func ResolveSecondarySSH(o SecondaryOpts) (mode string, port int, key string, err error) {
	mode = strings.ToLower(strings.TrimSpace(o.SSHMode))
	hasPass := strings.TrimSpace(o.SecondaryPass) != ""
	hasKey := strings.TrimSpace(o.SecondarySSHKey) != ""
	switch mode {
	case "", SSHModeInitial:
		if mode == "" {
			if hasKey && !hasPass {
				mode = SSHModeResume
			} else if hasPass && !hasKey {
				mode = SSHModeInitial
			} else if hasPass && hasKey {
				return "", 0, "", fmt.Errorf("secondary password and SSH key both set — set ssh_mode=initial or resume")
			} else {
				return "", 0, "", fmt.Errorf("secondary password (initial) or SSH key (resume) required")
			}
		}
	case SSHModeResume:
	default:
		return "", 0, "", fmt.Errorf("ssh_mode %q: want initial or resume", o.SSHMode)
	}
	if mode == SSHModeResume {
		key = strings.TrimSpace(o.SecondarySSHKey)
		if key == "" {
			key = strings.TrimSpace(o.PrimaryKey)
		}
		if key == "" {
			return "", 0, "", fmt.Errorf("resume deploy needs secondary SSH key (or primary key)")
		}
		return mode, 52222, key, nil
	}
	if !hasPass {
		return "", 0, "", fmt.Errorf("initial deploy needs secondary root password")
	}
	sep := strings.TrimSpace(o.SecondarySSHKey)
	if sep != "" && sep != strings.TrimSpace(o.PrimaryKey) {
		return "", 0, "", fmt.Errorf("initial deploy installs the primary operator key; a separate secondary_ssh_key is only valid in resume")
	}
	return SSHModeInitial, 22, "", nil
}

// secondaryDialKey is the key for every SSH/SCP to the secondary host.
// Primary connections stay on PrimaryKey. Same key may share the primary passphrase.
func secondaryDialKey(o SecondaryOpts) (path, pass string) {
	path = strings.TrimSpace(o.SecondarySSHKey)
	if path == "" {
		path = o.PrimaryKey
	}
	if path == o.PrimaryKey {
		pass = o.PrimaryKeyPassphrase
	}
	return
}

func secondaryKeyPass(o SecondaryOpts) string {
	if o.SecondarySSHKey == "" || o.SecondarySSHKey == o.PrimaryKey {
		return o.PrimaryKeyPassphrase
	}
	return ""
}

func requirePublicIPv4(host string) error {
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return fmt.Errorf("secondary host must be a public IP (got %q); hostname is not accepted for peer-id/svc-paths", host)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Errorf("secondary host IPv6 is not supported for peer-id encoding yet (got %q)", host)
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() || nonPublicIPv4(ip4) {
		return fmt.Errorf("secondary host %q is not a public IPv4 (private, CGNAT, documentation, benchmark, or reserved)", host)
	}
	return nil
}

// nonPublicIPv4 covers special-purpose ranges IsPrivate/IsLoopback miss.
// 100.64/10 CGNAT, TEST-NET-1/2/3, 198.18/15 benchmark, 240/4 reserved.
func nonPublicIPv4(ip net.IP) bool {
	var blocked = []string{
		"100.64.0.0/10",
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"198.18.0.0/15",
		"240.0.0.0/4",
	}
	for _, c := range blocked {
		_, n, err := net.ParseCIDR(c)
		if err == nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

// peerMaterialSCP is the subset of svc-paths material JSON needed before SCP to secondary.
type peerMaterialSCP struct {
	PeerID  string `json:"peer_id"`
	Slot    int    `json:"slot"`
	AddrPSP string `json:"addr_p_sp"`
	AddrSSP string `json:"addr_s_sp"`
	AddrPPS string `json:"addr_p_ps"`
	AddrSPS string `json:"addr_s_ps"`
	IfaceSP string `json:"iface_sp"`
	IfacePS string `json:"iface_ps"`
	PortSP  int    `json:"port_sp"`
	PortPS  int    `json:"port_ps"`
	WSSSP   int    `json:"wss_sp"`
	WSSPS   int    `json:"wss_ps"`
}

// materialJSONForSCP overlays validated slot defaults onto the original material.
// peerMaterialSCP has no WireGuard keys; those stay from raw.
func materialJSONForSCP(raw string, mat peerMaterialSCP) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		obj = map[string]any{}
	}
	obj["peer_id"] = mat.PeerID
	obj["slot"] = mat.Slot
	obj["addr_p_sp"] = mat.AddrPSP
	obj["addr_s_sp"] = mat.AddrSSP
	obj["addr_p_ps"] = mat.AddrPPS
	obj["addr_s_ps"] = mat.AddrSPS
	obj["iface_sp"] = mat.IfaceSP
	obj["iface_ps"] = mat.IfacePS
	obj["port_sp"] = mat.PortSP
	obj["port_ps"] = mat.PortPS
	obj["wss_sp"] = mat.WSSSP
	obj["wss_ps"] = mat.WSSPS
	return json.MarshalIndent(obj, "", "  ")
}

// validatePeerMaterialForSCP fills slot-0 defaults and refuses incomplete or wrong ports/addrs.
func validatePeerMaterialForSCP(mat *peerMaterialSCP, peerID string) error {
	if mat.PeerID != peerID {
		return fmt.Errorf("peer material identity: PeerID %q != expected %q — refuse SCP/bootstrap", mat.PeerID, peerID)
	}
	if mat.Slot < 0 || mat.Slot > 3 {
		return fmt.Errorf("peer material slot %d out of range", mat.Slot)
	}
	slot := svcpaths.SlotTable()[mat.Slot]
	if mat.Slot == 0 {
		if mat.IfaceSP == "" {
			mat.IfaceSP = slot.IfaceSP
		}
		if mat.IfacePS == "" {
			mat.IfacePS = slot.IfacePS
		}
		if mat.AddrPSP == "" {
			mat.AddrPSP = slot.AddrP_SP
		}
		if mat.AddrPPS == "" {
			mat.AddrPPS = slot.AddrP_PS
		}
		if mat.AddrSSP == "" {
			mat.AddrSSP = slot.AddrS_SP
		}
		if mat.AddrSPS == "" {
			mat.AddrSPS = slot.AddrS_PS
		}
		if mat.PortSP == 0 {
			mat.PortSP = slot.PortSP
		}
		if mat.PortPS == 0 {
			mat.PortPS = slot.PortPS
		}
		if mat.WSSSP == 0 {
			mat.WSSSP = slot.WSS_SP
		}
		if mat.WSSPS == 0 {
			mat.WSSPS = slot.WSS_PS
		}
	} else {
		if mat.IfaceSP == "" || mat.IfacePS == "" || mat.AddrPSP == "" || mat.AddrPPS == "" || mat.AddrSSP == "" || mat.AddrSPS == "" ||
			mat.PortSP == 0 || mat.PortPS == 0 || mat.WSSSP == 0 || mat.WSSPS == 0 {
			return fmt.Errorf("peer material slot %d incomplete (iface/addr/port/wss required) — refuse SCP", mat.Slot)
		}
	}
	if mat.PortSP != slot.PortSP || mat.PortPS != slot.PortPS || mat.WSSSP != slot.WSS_SP || mat.WSSPS != slot.WSS_PS {
		return fmt.Errorf("peer material slot %d ports/wss mismatch table (got port_sp=%d port_ps=%d wss_sp=%d wss_ps=%d want %d/%d/%d/%d) — refuse SCP",
			mat.Slot, mat.PortSP, mat.PortPS, mat.WSSSP, mat.WSSPS, slot.PortSP, slot.PortPS, slot.WSS_SP, slot.WSS_PS)
	}
	if mat.IfaceSP != slot.IfaceSP || mat.IfacePS != slot.IfacePS ||
		mat.AddrPSP != slot.AddrP_SP || mat.AddrSSP != slot.AddrS_SP || mat.AddrPPS != slot.AddrP_PS || mat.AddrSPS != slot.AddrS_PS {
		return fmt.Errorf("peer material slot %d iface/addr mismatch SlotTable — refuse SCP", mat.Slot)
	}
	return nil
}

type secondaryPack struct {
	SchemaVersion   int             `json:"schema_version"`
	MinAgentVersion string          `json:"min_agent_version"`
	Bundle          json.RawMessage `json:"bundle"`
	AgentID         string          `json:"agent_id"`
	AgentToken      string          `json:"agent_token"`
	CoreURL         string          `json:"core_url"`
	MTLSCAB64       string          `json:"mtls_ca_b64"`
	MTLSCertB64     string          `json:"mtls_cert_b64"`
	MTLSKeyB64      string          `json:"mtls_key_b64"`
	SNI             string          `json:"sni"`
}

// extractJSONObject returns the first complete top-level JSON object from mixed stdout.
// R12: no legacy "from { to EOF" fallback — decode must succeed.
func extractJSONObject(s string) (string, error) {
	i := strings.Index(s, "{")
	if i < 0 {
		return "", fmt.Errorf("no JSON object in output")
	}
	s = s[i:]
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return "", fmt.Errorf("JSON decode: %w", err)
	}
	return string(raw), nil
}

// materialIdentityPubs extracts stable public identity fields from ship JSON for verify on B.
func materialIdentityPubs(ship []byte) (spPubP, psPubP string) {
	var obj map[string]any
	if json.Unmarshal(ship, &obj) != nil {
		return "", ""
	}
	get := func(k string) string {
		v, _ := obj[k].(string)
		return strings.TrimSpace(v)
	}
	return get("sp_pub_p"), get("ps_pub_p")
}

// materialVerifyRemoteCmd is the remote script for full material compare on B.
// Expected JSON is supplied only via SSH stdin (never argv, never a file on B).
func materialVerifyRemoteCmd() string {
	return `set -e
f=/etc/netductor/svc-paths/material.json
test -f "$f"
if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 required for material verify — refuse skip-SCP" >&2
  exit 2
fi
python3 -c '
import json,sys
a=json.load(open(sys.argv[1]))
b=json.load(sys.stdin)
req=["peer_id","slot","sp_pub_p","ps_pub_p","sp_priv_s","ps_priv_s","sp_pub_s","ps_pub_s",
     "addr_p_sp","addr_s_sp","addr_p_ps","addr_s_ps","port_sp","port_ps","wss_sp","wss_ps","iface_sp","iface_ps"]
missing_e=[k for k in req if k not in b]
missing_i=[k for k in req if k not in a]
if missing_e:
    raise SystemExit("expected material missing fields: "+",".join(missing_e))
if missing_i:
    raise SystemExit("installed material missing fields: "+",".join(missing_i))
# Strict JSON equality (sort_keys + type-preserving dumps avoids True==1 pitfalls on diagnostics).
sa=json.dumps(a, sort_keys=True, separators=(",",":"))
sb=json.dumps(b, sort_keys=True, separators=(",",":"))
if sa!=sb:
    def walk(x,y,p):
        if type(x) is not type(y):
            yield p or "$"
            return
        if isinstance(x, dict):
            for k in sorted(set(x)|set(y)):
                np=(p+"." if p else "")+str(k)
                if k not in x:
                    yield np+"(extra)"; continue
                if k not in y:
                    yield np+"(missing)"; continue
                yield from walk(x[k],y[k],np)
            return
        if isinstance(x, list):
            if len(x)!=len(y):
                yield (p or "$")+"(len)"; return
            for i,(xi,yi) in enumerate(zip(x,y)):
                yield from walk(xi,yi,(p or "$")+"["+str(i)+"]")
            return
        if x!=y:
            yield p or "$"
    diffs=list(walk(a,b,""))
    raise SystemExit("mismatch fields: "+",".join(diffs) if diffs else "canonical")
print("material full match")
' "$f"
`
}

// DeploySecondary: Mac orchestrates both legs; primary never SSHs to secondary.

func DeploySecondary(o SecondaryOpts) error {
	if o.Promote {
		return fmt.Errorf("deploy does not promote: run client traffic test on B, then svc-paths peer ready/promote")
	}
	if o.SecondaryHost == "" {
		return fmt.Errorf("secondary host required")
	}
	return withHostDeployLock(o.SecondaryHost, func() error {
		return deploySecondaryBody(o)
	})
}

func deploySecondaryBody(o SecondaryOpts) error {
	if o.PrimaryHost == "" || o.PrimaryKey == "" {
		return fmt.Errorf("primary host and SSH key required")
	}
	if o.SecondaryHost == "" {
		return fmt.Errorf("secondary host required")
	}
	// Service-plane peer-ip requires a public IPv4 (not hostname). Fail before SSH harden.
	if errIP := requirePublicIPv4(o.SecondaryHost); errIP != nil {
		return errIP
	}
	mode, provPort, secKey, errMode := ResolveSecondarySSH(o)
	if errMode != nil {
		return errMode
	}
	o.SSHMode = mode
	if mode == SSHModeResume {
		o.SecondaryPass = "" // key-only; do not offer password
		o.SecondarySSHKey = secKey
	} else {
		o.SecondarySSHKey = "" // initial installs primary pubkey, not a separate key
	}
	if o.PrimaryUser == "" {
		o.PrimaryUser = "root"
	}
	if o.SecondaryUser == "" {
		o.SecondaryUser = "root"
	}
	if o.SNI == "" {
		o.SNI = ndconfig.DefaultSNI()
	}
	o.PrimaryKey = expandUserPath(o.PrimaryKey)
	o.SecondarySSHKey = expandUserPath(o.SecondarySSHKey)
	pub := strings.TrimSpace(o.OperatorPubKey)
	if pub == "" {
		pubPath := o.PrimaryKey + ".pub"
		b, err := os.ReadFile(pubPath)
		if err != nil {
			return fmt.Errorf("read operator pubkey %s: %w", pubPath, err)
		}
		pub = strings.TrimSpace(string(b))
	} else if !strings.HasPrefix(pub, "ssh-") {
		pubPath := expandUserPath(pub)
		b, err := os.ReadFile(pubPath)
		if err != nil {
			return fmt.Errorf("read operator pubkey %s: %w", pubPath, err)
		}
		pub = strings.TrimSpace(string(b))
	}
	if pub == "" {
		return fmt.Errorf("operator pubkey empty")
	}

	fmt.Fprintln(os.Stderr, "==> secondary deploy: Mac-direct (primary prepares pack; Mac SSHs to secondary)")
	pPort, errP := primarySSHPort(o)
	if errP != nil {
		return errP
	}
	prev, errState := loadDeployState(o.SecondaryHost)
	if errState != nil {
		return fmt.Errorf("deploy journal: %w", errState)
	}
	if prev != nil && effectiveStage(prev) != "" {
		fmt.Fprintf(os.Stderr, "==> resume deploy host=%s last_ok=%s status=%s peer=%s agent=%s\n",
			o.SecondaryHost, effectiveStage(prev), prev.Status, prev.PeerID, prev.AgentID)
	}

	peerIDEarly := strings.TrimSpace(o.PeerID)
	if peerIDEarly == "" {
		peerIDEarly = "peer-" + strings.ReplaceAll(o.SecondaryHost, ".", "-")
	}
	if prev != nil && prev.PeerID != "" {
		if strings.TrimSpace(o.PeerID) != "" && prev.PeerID != peerIDEarly {
			return fmt.Errorf("deploy journal peer_id %q != requested %q — refuse auto-switch", prev.PeerID, peerIDEarly)
		}
		peerIDEarly = prev.PeerID
	}
	var pack secondaryPack
	var out string
	var err error
	var raw string
	var errJ error

	// Advanced journal: never auto-reprovision. SSH/resume problems fail closed.
	advanced := canSkipProvision(prev) && prev != nil && prev.AgentID != ""
	skipProv := false
	if advanced {
		if o.SSHMode != SSHModeResume {
			return fmt.Errorf("deploy journal last_ok=%s agent=%s — refuse full provision in initial SSH mode; use resume (key :52222) or deploy journal clear %s",
				effectiveStage(prev), prev.AgentID, o.SecondaryHost)
		}
		fmt.Fprintln(os.Stderr, "==> skip prepare-pack+provision (journal stage>=provisioned, resume SSH)")
		pack.AgentID = prev.AgentID
		const day2PortProbe = "52222"
		secKeyP, secPassP := secondaryDialKey(o)
		if _, errP := runSSHOnPort(day2PortProbe, "", secKeyP, o.SecondaryUser, o.SecondaryHost, "true", secPassP); errP != nil {
			return failDeploy(&prev, peerIDEarly, prev.AgentID, o.SecondaryHost, fmt.Errorf("resume SSH %s:52222 failed (journal last_ok=%s) — refuse automatic re-provision: %w",
				o.SecondaryHost, effectiveStage(prev), errP))
		}
		skipProv = true
	}

	if !skipProv {
		if err := markCurrentStep(&prev, peerIDEarly, "", o.SecondaryHost, StageProvisionStarted); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "==> 1/2 primary prepare-pack on", o.PrimaryHost, "port", pPort)

		prep := "netductor secondary prepare-pack --sni " + ShellQuote(o.SNI)
		out, err = runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, prep, o.PrimaryKeyPassphrase)
		if err != nil {
			return failDeploy(&prev, peerIDEarly, "", o.SecondaryHost, fmt.Errorf("prepare-pack on primary: %w", err))
		}
		raw, errJ = extractJSONObject(out)
		if errJ != nil {
			return failDeploy(&prev, peerIDEarly, "", o.SecondaryHost, fmt.Errorf("prepare-pack JSON: %w", errJ))
		}
		if err := json.Unmarshal([]byte(raw), &pack); err != nil {
			return failDeploy(&prev, peerIDEarly, "", o.SecondaryHost, fmt.Errorf("parse prepare-pack JSON: %w", err))
		}
		if pack.AgentID == "" || pack.AgentToken == "" || len(pack.Bundle) == 0 {
			return failDeploy(&prev, peerIDEarly, "", o.SecondaryHost, fmt.Errorf("prepare-pack missing agent_id/token/bundle"))
		}
		const packSchema = 1
		if pack.SchemaVersion != packSchema {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("prepare-pack schema_version %d unsupported (need %d; missing/0 rejected)", pack.SchemaVersion, packSchema))
		}
		if pack.MinAgentVersion != "" {
			if !version.ValidDotVersion(pack.MinAgentVersion) {
				return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("prepare-pack min_agent_version %q invalid", pack.MinAgentVersion))
			}
			if version.Less(version.Release, pack.MinAgentVersion) {
				return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("operator binary %s older than pack min_agent_version %s — upgrade before provision", version.Release, pack.MinAgentVersion))
			}
		}
		ca, err := base64.StdEncoding.DecodeString(pack.MTLSCAB64)
		if err != nil {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("mtls ca: %w", err))
		}
		cert, err := base64.StdEncoding.DecodeString(pack.MTLSCertB64)
		if err != nil {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("mtls cert: %w", err))
		}
		key, err := base64.StdEncoding.DecodeString(pack.MTLSKeyB64)
		if err != nil {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("mtls key: %w", err))
		}

		if o.SSHMode == SSHModeResume {
			fmt.Fprintln(os.Stderr, "==> 2/2 Mac → secondary", o.SecondaryHost, "(resume, key-only :52222)")
		} else {
			fmt.Fprintln(os.Stderr, "==> 2/2 Mac → secondary", o.SecondaryHost, "(initial, password once, then key-only)")
		}
		// R6/R16: host-key mismatch must NOT auto-forget or retry with password (MITM risk).
		// Explicit ForceReinstall only after operator verified fingerprint offline.
		res, err := secondary.ProvisionFromCore(secondary.ProvisionIn{
			Host: o.SecondaryHost, Port: provPort, User: o.SecondaryUser,
			Password: o.SecondaryPass, SSHPrivateKey: o.SecondarySSHKey,
			SSHKeyPassphrase: secondaryKeyPass(o),
			SNI:              o.SNI, OperatorPubKey: pub,
			MTLSCA: ca, MTLSCert: cert, MTLSKey: key,
			ForceReinstall: o.ForceReinstall,
		}, string(pack.Bundle))
		if err != nil && isSSHHostKeyError(err) {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("SSH host key mismatch for %s — refuse auto-trust.\n"+
				"Verify fingerprint offline, then:\n"+
				"  netductor ssh-hosts forget --kind relay %s\n"+
				"  ssh-keygen -R %s; ssh-keygen -R '[%s]:22'; ssh-keygen -R '[%s]:52222'\n"+
				"Re-run deploy (or set ForceReinstall after forget).\n%w",
				o.SecondaryHost, o.SecondaryHost, o.SecondaryHost, o.SecondaryHost, o.SecondaryHost, err))
		}
		if res != nil {
			fmt.Print(res.Log)
		}
		if err != nil {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("secondary provision: %w", err))
		}
		if res == nil || !res.AgentOK || !res.SingBoxOK {
			return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("secondary provision: agent/sing-box not healthy after join"))
		}
	} // end !skipProv

	const day2Port = "52222"
	fmt.Fprintln(os.Stderr, "==> verify SSH key-only on", o.SecondaryHost+":"+day2Port)
	secKey, secPass := secondaryDialKey(o)
	if _, err := runSSHOnPort(day2Port, "", secKey, o.SecondaryUser, o.SecondaryHost, "true", secPass); err != nil {
		return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("post-harden SSH %s:%s failed (key-only): %w", o.SecondaryHost, day2Port, err))
	}
	if err := checkpointSuccess(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, StageProvisioned); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "==> primary: secondary sync + fleet status")
	out, err = runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
		"netductor secondary sync && netductor fleet status",
		o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return failDeploy(&prev, peerIDEarly, pack.AgentID, o.SecondaryHost, fmt.Errorf("primary secondary sync: %w", err))
	}

	if path, err := CollectOperatorSecrets("secondary", o.SecondaryUser, o.SecondaryHost, secKey, secPass); err != nil {
		fmt.Fprintln(os.Stderr, "warn: credentials collect:", err)
	} else {
		fmt.Fprintln(os.Stderr, "  Credentials file:", path)
	}

	fmt.Fprintln(os.Stderr, "==> 3/3 svc-paths peer add + bootstrap (Blue-Green slot)")
	primIP, secIP := o.PrimaryHost, o.SecondaryHost
	peerID := peerIDEarly
	if canSkipPeerAdd(prev) {
		fmt.Fprintln(os.Stderr, "==> peer add resume: expect existing material (idempotent PeerAdd, no new keys)")
	}
	if err := markCurrentStep(&prev, peerID, pack.AgentID, o.SecondaryHost, StagePeerAddStarted); err != nil {
		return err
	}
	pre := fmt.Sprintf(
		"set -e; mkdir -p /etc/netductor/secrets; printf '%%s\\n' %s > /etc/netductor/secrets/public_ip; netductor svc-paths peer add --peer-ip %s --id %s --primary-ip %s",
		ShellQuote(primIP), ShellQuote(secIP), ShellQuote(peerID), ShellQuote(primIP),
	)
	out, err = runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, pre, o.PrimaryKeyPassphrase)
	if err != nil && (strings.Contains(out, "peer recover") || (err != nil && strings.Contains(err.Error(), "peer recover"))) {
		fmt.Fprintln(os.Stderr, "==> incomplete peer add — recovering", peerID)
		rec := "netductor svc-paths peer recover " + ShellQuote(peerID)
		outR, errR := runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, rec, o.PrimaryKeyPassphrase)
		fmt.Print(outR)
		if errR != nil {
			return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths peer recover: %w (original add: %v)", errR, err))
		}
		out, err = runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, pre, o.PrimaryKeyPassphrase)
	}
	if err != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths peer add: %w", err))
	}
	raw, errJ = extractJSONObject(out)
	if errJ != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("peer add material JSON: %w", errJ))
	}
	var mat peerMaterialSCP
	if err := json.Unmarshal([]byte(raw), &mat); err != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("peer material decode: %w", err))
	}
	if err := validatePeerMaterialForSCP(&mat, peerID); err != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, err)
	}
	if err := checkpointSuccess(&prev, peerID, pack.AgentID, o.SecondaryHost, StagePeerPrepared); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "  peer slot=%d ifaces=%s/%s primary_sp=%s wss_sp=%d port_sp=%d\n", mat.Slot, mat.IfaceSP, mat.IfacePS, mat.AddrPSP, mat.WSSSP, mat.PortSP)

	// Ship material stays in operator memory; applied on B via SSH stdin (no remote /tmp secret file).
	ship, errShip := materialJSONForSCP(raw, mat)
	if errShip != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths material normalize: %w", errShip))
	}
	sum := sha256.Sum256(ship)
	shipSHA := hex.EncodeToString(sum[:])

	skipSCP := canSkipMaterialXfer(prev) && prev != nil && prev.MaterialSHA != "" && prev.MaterialSHA == shipSHA
	if skipSCP {
		// Expected material only on SSH stdin (not argv, not /tmp). Field names only on mismatch.
		verifyMat := materialVerifyRemoteCmd()
		outV, errV := runSSHOnPortWithStdin(day2Port, "", secKey, o.SecondaryUser, o.SecondaryHost, verifyMat, secPass, ship)
		if errV != nil {
			fmt.Fprintln(os.Stderr, "warn: material on B does not match ship — re-SCP")
			for _, line := range strings.Split(outV, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				l := strings.ToLower(line)
				if strings.Contains(l, "priv") || strings.Contains(l, "secret") || strings.Contains(l, "token") || strings.Contains(l, "key") {
					continue
				}
				if strings.HasPrefix(line, "mismatch ") || strings.HasPrefix(line, "expected ") || strings.HasPrefix(line, "installed ") || strings.HasPrefix(line, "python3 required") {
					fmt.Fprintln(os.Stderr, "  remote:", line)
				}
			}
			skipSCP = false
			if prev != nil {
				prev.BootstrappedMaterialSHA = ""
			}
		} else {
			fmt.Fprintln(os.Stderr, strings.TrimSpace(outV))
			fmt.Fprintln(os.Stderr, "==> skip SCP (full material match + journal SHA)")
		}
	}
	if !skipSCP {
		fmt.Fprintln(os.Stderr, "==> material will apply via bootstrap-secondary stdin (no remote temp file)")
	}
	if err := checkpointSuccessWithSHA(&prev, peerID, pack.AgentID, o.SecondaryHost, StageMaterialTransferred, shipSHA); err != nil {
		return err
	}

	// Slot-aware verification on B (not hardcoded slot 0).
	secVerify := fmt.Sprintf(`set -e
test -f /etc/netductor/secrets/secondary_core_url
grep -q %s /etc/netductor/secrets/secondary_core_url
test -f /etc/netductor/svc-paths/material.json
grep -q '"peer_id"[[:space:]]*:[[:space:]]*"%s"' /etc/netductor/svc-paths/material.json
grep -q '"slot"[[:space:]]*:[[:space:]]*%d' /etc/netductor/svc-paths/material.json
for u in nd-wss-sp-client nd-wss-ps-server wg-quick@%s wg-quick@%s; do
  st=$(systemctl is-active "$u" 2>/dev/null || true)
  test "$st" = active
done
ping -c1 -W3 %s >/dev/null
ping -c1 -W3 %s >/dev/null
echo "svc-paths secondary verify ok slot_iface=%s/%s"
`, ShellQuote(mat.AddrPSP), peerID, mat.Slot, ShellQuote(mat.IfaceSP), ShellQuote(mat.IfacePS),
		ShellQuote(mat.AddrPSP), ShellQuote(mat.AddrPPS), ShellQuote(mat.IfaceSP), ShellQuote(mat.IfacePS))
	secBoot := fmt.Sprintf(`set -e
netductor svc-paths bootstrap-secondary --material -
test -f /etc/netductor/secrets/secondary_core_url
grep -q %s /etc/netductor/secrets/secondary_core_url
for u in nd-wss-sp-client nd-wss-ps-server wg-quick@%s wg-quick@%s; do
  st=$(systemctl is-active "$u" 2>/dev/null || true)
  test "$st" = active
done
ping -c1 -W3 %s >/dev/null
ping -c1 -W3 %s >/dev/null
echo "svc-paths secondary ok slot_iface=%s/%s"
`, ShellQuote(mat.AddrPSP), ShellQuote(mat.IfaceSP), ShellQuote(mat.IfacePS),
		ShellQuote(mat.AddrPSP), ShellQuote(mat.AddrPPS), ShellQuote(mat.IfaceSP), ShellQuote(mat.IfacePS))
	useStdinShip := true
	bootScript := secBoot
	if canSkipBootstrapForMaterial(prev, shipSHA) {
		fmt.Fprintln(os.Stderr, "==> skip bootstrap-secondary (journal+material SHA match) — verify only")
		bootScript = secVerify
		useStdinShip = false
	} else if canSkipBootstrapB(prev) {
		fmt.Fprintln(os.Stderr, "==> material changed or bootstrap not bound to ship SHA — full bootstrap via stdin")
	}

	var out2 string
	var err2 error
	if useStdinShip {
		out2, err2 = runSSHOnPortWithStdin(day2Port, "", secKey, o.SecondaryUser, o.SecondaryHost, bootScript, secPass, ship)
	} else {
		out2, err2 = runSSHOnPort(day2Port, "", secKey, o.SecondaryUser, o.SecondaryHost, bootScript, secPass)
	}
	if err2 != nil {
		if o.SSHMode != SSHModeResume && day2Port == "52222" {
			var out2b string
			var err2b error
			if useStdinShip {
				out2b, err2b = runSSHOnPortWithStdin("22", "", secKey, o.SecondaryUser, o.SecondaryHost, bootScript, secPass, ship)
			} else {
				out2b, err2b = runSSHOnPort("22", "", secKey, o.SecondaryUser, o.SecondaryHost, bootScript, secPass)
			}
			if err2b != nil {
				return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths bootstrap-secondary (SP/PS strict): %w", err2b))
			}
			out2 = out2b
		} else {
			return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths bootstrap-secondary (SP/PS strict): %w", err2))
		}
	}
	fmt.Fprintln(os.Stderr, strings.TrimSpace(out2))
	if err := checkpointBootstrapOK(&prev, peerID, pack.AgentID, o.SecondaryHost, shipSHA); err != nil {
		return err
	}

	// Primary firewall for this peer: mTLS 8789 from secondary WG IPs + WSS_SP listen.
	// Fail deploy if ufw is active and rules cannot be applied.
	allowNoUFW := os.Getenv("NETDUCTOR_ALLOW_NO_UFW")
	ufwPeer := fmt.Sprintf(`set -e
export NETDUCTOR_ALLOW_NO_UFW=%s
if ! command -v ufw >/dev/null 2>&1 || ! ufw status 2>/dev/null | grep -qi 'Status: active'; then
  if [ "${NETDUCTOR_ALLOW_NO_UFW:-}" = "1" ]; then
    echo "ufw inactive or missing - skip (NETDUCTOR_ALLOW_NO_UFW=1)"
    exit 0
  fi
  echo "ufw inactive or missing - refuse (set NETDUCTOR_ALLOW_NO_UFW=1 to override)" >&2
  exit 1
fi
ufw allow from %s to any port 8789 proto tcp comment nd-mtls-sp
ufw allow from %s to any port 8789 proto tcp comment nd-mtls-ps
ufw allow from %s to any port %d proto tcp comment nd-wss-sp-slot
status=$(ufw status)
need() {
  src="$1"; port="$2"
  # Require same line: ALLOW IN + source IP + port/tcp.
  echo "$status" | awk -v src="$src" -v port="$port/tcp" '
    /ALLOW[[:space:]]+IN/ {
      s=0; p=0
      for (i=1;i<=NF;i++) { if ($i==src) s=1; if ($i==port) p=1 }
      if (s && p) found=1
    }
    END { exit found?0:1 }'
}
need %s 8789
need %s 8789
need %s %d
echo "ufw peer rules verified"
`, ShellQuote(allowNoUFW), ShellQuote(mat.AddrSSP), ShellQuote(mat.AddrSPS), ShellQuote(secIP), mat.WSSSP,
		ShellQuote(mat.AddrSSP), ShellQuote(mat.AddrSPS), ShellQuote(secIP), mat.WSSSP)
	if outU, errU := runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, ufwPeer, o.PrimaryKeyPassphrase); errU != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths primary firewall for peer slot: %w", errU))
	} else {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(outU))
	}
	if err := checkpointSuccess(&prev, peerID, pack.AgentID, o.SecondaryHost, StageFirewallVerified); err != nil {
		return err
	}

	// Primary: units active + ping secondary WG endpoints (bidirectional with B's pings to primary).
	spUnit := "nd-wss-sp-server"
	if mat.IfaceSP != "nd-svc-sp" {
		spUnit = "nd-wss-sp-server-" + mat.IfaceSP
	}
	psUnit := "nd-wss-ps-client"
	if mat.IfacePS != "nd-svc-ps" {
		psUnit = "nd-wss-ps-client-" + mat.IfacePS
	}
	primCheck := fmt.Sprintf(`set -e
for u in %s %s wg-quick@%s wg-quick@%s; do
  st=$(systemctl is-active "$u" 2>/dev/null || true)
  test "$st" = active || { echo "unit $u=$st"; exit 1; }
done
ping -c1 -W3 %s >/dev/null
ping -c1 -W3 %s >/dev/null
echo "primary peer units+ping ok -> %s %s"
`, ShellQuote(spUnit), ShellQuote(psUnit), ShellQuote(mat.IfaceSP), ShellQuote(mat.IfacePS),
		ShellQuote(mat.AddrSSP), ShellQuote(mat.AddrSPS), ShellQuote(mat.AddrSSP), ShellQuote(mat.AddrSPS))
	if outP2, errP2 := runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, primCheck, o.PrimaryKeyPassphrase); errP2 != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("svc-paths primary SP/PS units/ping: %v", errP2))
	} else {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(outP2))
	}
	if err := checkpointSuccess(&prev, peerID, pack.AgentID, o.SecondaryHost, StageNetworkVerified); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "==> wait secondary online on primary (heartbeat)")
	if pack.AgentID == "" {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("prepare-pack missing agent_id — refuse to treat any online secondary as this deploy"))
	}
	// Exact AgentID + online + fresh last_seen (seen= unix, within 90s) after SP switch.
	waitCmd := fmt.Sprintf(`ok=0
id=%s
now=$(date +%%s)
for i in $(seq 1 18); do
  st=$(netductor secondary status 2>/dev/null || true)
  echo "$st"
  if printf '%%s\n' "$st" | awk -F '\t' -v id="$id" -v now="$now" '
    $1 == id && $3 == "online" {
      for (i=1;i<=NF;i++) if ($i ~ /^seen=/) {
        split($i, a, "="); age = now - a[2];
        if (age >= 0 && age <= 90) found=1
      }
    }
    END { exit !found }'; then
    ok=1
    break
  fi
  sleep 5
  now=$(date +%%s)
done
test "$ok" = 1`, ShellQuote(pack.AgentID))
	out, err = runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, waitCmd, o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("secondary not online on primary after deploy (agent heartbeat missing): %w", err))
	}

	if err := checkpointSuccess(&prev, peerID, pack.AgentID, o.SecondaryHost, StageHeartbeatVerified); err != nil {
		return err
	}

	// Profile readiness: B must be subscription-eligible (PBK/SID/PublicIP/sing-box).
	fmt.Fprintln(os.Stderr, "==> check secondary profile readiness (PBK/SID/sing-box)")
	readyCmd := fmt.Sprintf(`set -e
id=%s
ip=%s
f=/var/lib/netductor/secondary/devices.json
test -f "$f"
python3 -c '
import json,sys
id,ip=sys.argv[1],sys.argv[2]
r=json.load(open("/var/lib/netductor/secondary/devices.json"))
devs=r.get("devices") or []
d=next((x for x in devs if x.get("id")==id), None)
if not d:
    raise SystemExit("agent id not in registry")
if (d.get("public_ip") or "") != ip:
    raise SystemExit("public_ip mismatch")
if not (d.get("pbk") or "").strip():
    raise SystemExit("empty pbk — subscription would omit this secondary")
if not (d.get("sid") or "").strip():
    raise SystemExit("empty sid — subscription would omit this secondary")
if not d.get("singbox_ok"):
    raise SystemExit("singbox_ok=false")
print("profile ready id=%%s ip=%%s pbk_len=%%d"%%(id,ip,len(d.get("pbk") or "")))
' "$id" "$ip"
`, ShellQuote(pack.AgentID), ShellQuote(o.SecondaryHost))
	outReady, errReady := runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, readyCmd, o.PrimaryKeyPassphrase)
	if errReady != nil {
		return failDeploy(&prev, peerID, pack.AgentID, o.SecondaryHost, fmt.Errorf("secondary profile not subscription-ready: %w", errReady))
	}
	fmt.Fprintln(os.Stderr, strings.TrimSpace(outReady))

	// DNS cutover requires shared VPN hostname — warn only (deploy does not cut over DNS).
	hostNote := `set -e
h=""
if [ -n "${NETDUCTOR_VPN_HOST:-}" ]; then h="$NETDUCTOR_VPN_HOST"; fi
if [ -z "$h" ] && [ -f /etc/netductor/vpn_hostname ]; then h=$(tr -d "\n" </etc/netductor/vpn_hostname); fi
if [ -z "$h" ]; then
  echo "NOTE: no NETDUCTOR_VPN_HOST / vpn_hostname — client profiles may use direct secondary IP; DNS cutover alone will not switch them"
else
  echo "vpn_hostname=$h (DNS cutover targets this name; profiles use B Reality keys after promote)"
fi
`
	if outH, errH := runSSHOnPort(pPort, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, hostNote, o.PrimaryKeyPassphrase); errH == nil {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(outH))
	}

	if err := checkpointDeployComplete(&prev, peerID, pack.AgentID, o.SecondaryHost); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "==> not promoted (after traffic test: peer validate/ready/promote", peerID+")")

	fmt.Fprintf(os.Stderr, "    note: public surface may include :443 (Reality), :52222 (SSH), :%d (PS WSS)\n", mat.WSSPS)
	fmt.Fprintln(os.Stderr, "    if vpn_hostname is set, update DNS A to", o.SecondaryHost, "before clients reconnect")
	return nil
}

func isSSHHostKeyError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "host key") ||
		strings.Contains(s, "known_hosts") ||
		strings.Contains(s, "remote host identification has changed") ||
		strings.Contains(s, "someone is doing something nasty")
}

func expandUserPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func primarySSHPort(o SecondaryOpts) (string, error) {
	port := o.PrimarySSHPort
	if port == 0 {
		port = 52222
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("primary SSH port %d invalid (need 1-65535; 0 means default)", port)
	}
	return fmt.Sprintf("%d", port), nil
}
