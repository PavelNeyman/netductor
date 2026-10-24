package deploy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/secondary"
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
	SecondaryHost        string
	SecondaryUser        string
	SecondaryPass        string
	SecondarySSHKey      string // optional re-provision when password already off
	SNI                  string
	OperatorPubKey       string
	// ForceReinstall: operator already verified host-key rotation — pass through to Go TOFU forget.
	ForceReinstall bool
	// Promote writes preferred_secondary_ip (Blue-Green: leave false until peer promote).
	Promote bool
	// ViaPrimary is ignored (deprecated). Always Mac-direct.
	ViaPrimary bool
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

// DeploySecondary: Mac orchestrates both legs; primary never SSHs to secondary.
func DeploySecondary(o SecondaryOpts) error {
	if o.PrimaryHost == "" || o.PrimaryKey == "" {
		return fmt.Errorf("primary host and SSH key required")
	}
	if o.SecondaryHost == "" {
		return fmt.Errorf("secondary host required")
	}
	if o.SecondaryPass == "" && o.SecondarySSHKey == "" {
		return fmt.Errorf("secondary password or secondary SSH key required")
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
	fmt.Fprintln(os.Stderr, "==> 1/2 primary prepare-pack on", o.PrimaryHost)

	prep := "netductor secondary prepare-pack --sni " + ShellQuote(o.SNI)
	out, err := runSSHOnPort("52222", "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, prep, o.PrimaryKeyPassphrase)
	if err != nil {
		return fmt.Errorf("prepare-pack on primary: %w\n%s", err, out)
	}
	raw, errJ := extractJSONObject(out)
	if errJ != nil {
		return fmt.Errorf("prepare-pack JSON: %w\n%s", errJ, out)
	}
	var pack secondaryPack
	if err := json.Unmarshal([]byte(raw), &pack); err != nil {
		return fmt.Errorf("parse prepare-pack JSON: %w\n%s", err, out)
	}
	if pack.AgentID == "" || pack.AgentToken == "" || len(pack.Bundle) == 0 {
		return fmt.Errorf("prepare-pack missing agent_id/token/bundle:\n%s", out)
	}
	const packSchema = 1
	if pack.SchemaVersion != packSchema {
		return fmt.Errorf("prepare-pack schema_version %d unsupported (need %d; missing/0 rejected)", pack.SchemaVersion, packSchema)
	}
	ca, err := base64.StdEncoding.DecodeString(pack.MTLSCAB64)
	if err != nil {
		return fmt.Errorf("mtls ca: %w", err)
	}
	cert, err := base64.StdEncoding.DecodeString(pack.MTLSCertB64)
	if err != nil {
		return fmt.Errorf("mtls cert: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(pack.MTLSKeyB64)
	if err != nil {
		return fmt.Errorf("mtls key: %w", err)
	}

	fmt.Fprintln(os.Stderr, "==> 2/2 Mac → secondary", o.SecondaryHost, "(password once, then key-only)")
	// R6/R16: host-key mismatch must NOT auto-forget or retry with password (MITM risk).
	// Explicit ForceReinstall only after operator verified fingerprint offline.
	res, err := secondary.ProvisionFromCore(secondary.ProvisionIn{
		Host: o.SecondaryHost, Port: 22, User: o.SecondaryUser,
		Password: o.SecondaryPass, SSHPrivateKey: o.SecondarySSHKey,
		SNI: o.SNI, OperatorPubKey: pub,
		MTLSCA: ca, MTLSCert: cert, MTLSKey: key,
		ForceReinstall: o.ForceReinstall,
	}, string(pack.Bundle))
	if err != nil && isSSHHostKeyError(err) {
		return fmt.Errorf("SSH host key mismatch for %s — refuse auto-trust.\n"+
			"Verify fingerprint offline, then:\n"+
			"  netductor ssh-hosts forget --kind relay %s\n"+
			"  ssh-keygen -R %s; ssh-keygen -R '[%s]:22'; ssh-keygen -R '[%s]:52222'\n"+
			"Re-run deploy (or set ForceReinstall after forget).\n%w",
			o.SecondaryHost, o.SecondaryHost, o.SecondaryHost, o.SecondaryHost, o.SecondaryHost, err)
	}
	if res != nil {
		fmt.Print(res.Log)
	}
	if err != nil {
		return fmt.Errorf("secondary provision: %w", err)
	}
	if res == nil || !res.AgentOK || !res.SingBoxOK {
		return fmt.Errorf("secondary provision: agent/sing-box not healthy after join")
	}

	const day2Port = "52222"
	fmt.Fprintln(os.Stderr, "==> verify SSH key-only on", o.SecondaryHost+":"+day2Port)
	if out, err := runSSHOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, "true", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("post-harden SSH %s:%s failed (key-only): %w\n%s", o.SecondaryHost, day2Port, err, out)
	}

	fmt.Fprintln(os.Stderr, "==> primary: secondary sync + fleet status")
	out, err = runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
		"netductor secondary sync && netductor fleet status",
		o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return fmt.Errorf("primary secondary sync: %w\n%s", err, out)
	}

	if path, err := CollectOperatorSecrets("secondary", o.SecondaryUser, o.SecondaryHost, o.PrimaryKey, o.PrimaryKeyPassphrase); err != nil {
		fmt.Fprintln(os.Stderr, "warn: credentials collect:", err)
	} else {
		fmt.Fprintln(os.Stderr, "  Credentials file:", path)
	}

	fmt.Fprintln(os.Stderr, "==> 3/3 svc-paths bootstrap (SP/PS)")
	primIP, secIP := o.PrimaryHost, o.SecondaryHost
	pre := fmt.Sprintf(
		"mkdir -p /etc/netductor/secrets; printf '%%s\\n' %s > /etc/netductor/secrets/public_ip; netductor svc-paths bootstrap-primary --peer-ip %s",
		ShellQuote(primIP), ShellQuote(secIP),
	)
	out, err = runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, pre, o.PrimaryKeyPassphrase)
	if err != nil {
		return fmt.Errorf("svc-paths bootstrap-primary: %w\n%s", err, out)
	}
	raw, errJ = extractJSONObject(out)
	if errJ != nil {
		return fmt.Errorf("bootstrap-primary material JSON: %w\n%s", errJ, out)
	}
	tmpFile, errTmp := os.CreateTemp("", "nd-svc-material-*.json")
	if errTmp != nil {
		return fmt.Errorf("svc-paths temp file: %w", errTmp)
	}
	tmpPath := tmpFile.Name()
	if _, errW := tmpFile.Write([]byte(raw)); errW != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("svc-paths material write: %w", errW)
	}
	if errC := tmpFile.Close(); errC != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("svc-paths material close: %w", errC)
	}
	defer os.Remove(tmpPath)

	scpErr := runSCPOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, tmpPath, "/tmp/nd-svc-material.json", o.PrimaryKeyPassphrase)
	if scpErr != nil {
		fmt.Fprintln(os.Stderr, "warn svc-paths scp :52222:", scpErr, "— retry :22")
		scpErr = runSCPOnPort("22", "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, tmpPath, "/tmp/nd-svc-material.json", o.PrimaryKeyPassphrase)
	}
	if scpErr != nil {
		return fmt.Errorf("svc-paths scp material to secondary: %w", scpErr)
	}
	secBoot := "echo c2V0IC1lCnRyYXAgJ3JtIC1mIC90bXAvbmQtc3ZjLW1hdGVyaWFsLmpzb24nIEVYSVQKY2htb2QgNjAwIC90bXAvbmQtc3ZjLW1hdGVyaWFsLmpzb24KbmV0ZHVjdG9yIHN2Yy1wYXRocyBib290c3RyYXAtc2Vjb25kYXJ5IC0tbWF0ZXJpYWwgL3RtcC9uZC1zdmMtbWF0ZXJpYWwuanNvbgp0ZXN0IC1mIC9ldGMvbmV0ZHVjdG9yL3NlY3JldHMvc2Vjb25kYXJ5X2NvcmVfdXJsCmdyZXAgLXEgMTAuODcuMTAuMSAvZXRjL25ldGR1Y3Rvci9zZWNyZXRzL3NlY29uZGFyeV9jb3JlX3VybApmb3IgdSBpbiBuZC13c3Mtc3AtY2xpZW50IG5kLXdzcy1wcy1zZXJ2ZXIgd2ctcXVpY2tAbmQtc3ZjLXNwIHdnLXF1aWNrQG5kLXN2Yy1wczsgZG8KICBzdD0kKHN5c3RlbWN0bCBpcy1hY3RpdmUgIiR1IiAyPi9kZXYvbnVsbCB8fCBlY2hvIGluYWN0aXZlKQogIHRlc3QgIiRzdCIgPSAiYWN0aXZlIiB8fCB7IGVjaG8gInVuaXQgJHUgbm90IGFjdGl2ZTogJHN0IiA+JjI7IGV4aXQgMTsgfQpkb25lCnBpbmcgLWMxIC1XMyAxMC44Ny4xMC4xIHx8IHsgZWNobyAiU1AgcGVlciAxMC44Ny4xMC4xIHVucmVhY2hhYmxlIiA+JjI7IGV4aXQgMTsgfQpwaW5nIC1jMSAtVzMgMTAuODcuMTEuMSB8fCB7IGVjaG8gIlBTIHBlZXIgMTAuODcuMTEuMSB1bnJlYWNoYWJsZSIgPiYyOyBleGl0IDE7IH0KdGVzdCAtZiAvZXRjL25ldGR1Y3Rvci9zdmMtcGF0aHMvZmFpbG92ZXItcG9saWN5Lmpzb24KdGVzdCAiJChzeXN0ZW1jdGwgaXMtYWN0aXZlIG5kLXN2Yy1wYXRocy1oZWFsdGgudGltZXIgMj4vZGV2L251bGwgfHwgZWNobyBpbmFjdGl2ZSkiID0gImFjdGl2ZSIgfHwgeyBlY2hvICJuZC1zdmMtcGF0aHMtaGVhbHRoLnRpbWVyIG5vdCBhY3RpdmUiID4mMjsgZXhpdCAxOyB9CmVjaG8gU1BfUFNfT0sK | base64 -d | bash"
	out2, err2 := runSSHOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, secBoot, o.PrimaryKeyPassphrase)
	fmt.Print(out2)
	if err2 != nil {
		out2b, err2b := runSSHOnPort("22", "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, secBoot, o.PrimaryKeyPassphrase)
		fmt.Print(out2b)
		if err2b != nil {
			return fmt.Errorf("svc-paths bootstrap-secondary (SP/PS strict): %w\n%s", err2b, out2b)
		}
		out2 = out2b
	}
	if !strings.Contains(out2, "SP_PS_OK") {
		return fmt.Errorf("svc-paths secondary: SP/PS verification failed\n%s", out2)
	}
	primBoot := "echo c2V0IC1lCmZvciB1IGluIG5kLXdzcy1zcC1zZXJ2ZXIgbmQtd3NzLXBzLWNsaWVudCB3Zy1xdWlja0BuZC1zdmMtc3Agd2ctcXVpY2tAbmQtc3ZjLXBzOyBkbwogIHN0PSQoc3lzdGVtY3RsIGlzLWFjdGl2ZSAiJHUiIDI+L2Rldi9udWxsIHx8IGVjaG8gaW5hY3RpdmUpCiAgdGVzdCAiJHN0IiA9ICJhY3RpdmUiIHx8IHsgZWNobyAicHJpbWFyeSB1bml0ICR1IG5vdCBhY3RpdmU6ICRzdCIgPiYyOyBleGl0IDE7IH0KZG9uZQojIEJvdGggZGlyZWN0aW9uczogU1AgcGVlciBhbmQgUFMgcGVlciAoUjgpLgpwaW5nIC1jMSAtVzMgMTAuODcuMTAuMiB8fCB7IGVjaG8gIlNQIHBlZXIgMTAuODcuMTAuMiB1bnJlYWNoYWJsZSIgPiYyOyBleGl0IDE7IH0KcGluZyAtYzEgLVczIDEwLjg3LjExLjIgfHwgeyBlY2hvICJQUyBwZWVyIDEwLjg3LjExLjIgdW5yZWFjaGFibGUiID4mMjsgZXhpdCAxOyB9CnRlc3QgLWYgL2V0Yy9uZXRkdWN0b3Ivc3ZjLXBhdGhzL2ZhaWxvdmVyLXBvbGljeS5qc29uCnRlc3QgIiQoc3lzdGVtY3RsIGlzLWFjdGl2ZSBuZC1zdmMtcGF0aHMtaGVhbHRoLnRpbWVyIDI+L2Rldi9udWxsIHx8IGVjaG8gaW5hY3RpdmUpIiA9ICJhY3RpdmUiIHx8IHsgZWNobyAibmQtc3ZjLXBhdGhzLWhlYWx0aC50aW1lciBub3QgYWN0aXZlIiA+JjI7IGV4aXQgMTsgfQplY2hvIFBSSU1BUllfU1BfT0sK | base64 -d | bash"
	outP2, errP2 := runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, primBoot, o.PrimaryKeyPassphrase)
	fmt.Print(outP2)
	if errP2 != nil || !strings.Contains(outP2, "PRIMARY_SP_OK") {
		return fmt.Errorf("svc-paths primary SP/PS units: %v\n%s", errP2, outP2)
	}
	_, _ = runSSHOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost,
		`set -e; systemctl stop ssh.socket sshd.socket 2>/dev/null || true; systemctl disable ssh.socket sshd.socket 2>/dev/null || true; systemctl enable ssh.service 2>/dev/null || true; systemctl restart ssh 2>/dev/null || systemctl restart sshd 2>/dev/null || true`,
		o.PrimaryKeyPassphrase)
	// R10: key login on :22 must fail.
	if out22, err22 := runSSHOnPort("22", "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, "true", o.PrimaryKeyPassphrase); err22 == nil {
		return fmt.Errorf("SSH still reachable on :22 after harden (expected key-only :%s only)", day2Port)
	} else {
		msg := strings.ToLower(err22.Error() + " " + out22)
		switch {
		case strings.Contains(msg, "connection refused"), strings.Contains(msg, "no route"):
			fmt.Fprintln(os.Stderr, "==> :22 closed or unreachable (ok for harden)")
		case strings.Contains(msg, "permission denied"), strings.Contains(msg, "authentication"):
			fmt.Fprintln(os.Stderr, "==> :22 rejects key")
		default:
			fmt.Fprintln(os.Stderr, "==> :22 not usable with key:", strings.TrimSpace(err22.Error()))
		}
	}
	// Effective sshd: PasswordAuthentication must be no (sshd -T). :22 may still listen (key-only) — OK;
	// fail if password auth remains enabled.
	probe22 := `set -e
eff=$(sshd -T 2>/dev/null | awk '/^passwordauthentication /{print $2; exit}')
echo "passwordauthentication=$eff"
test "$eff" = "no" || { echo "effective PasswordAuthentication is not no: $eff" >&2; exit 1; }
if ss -tlnp 2>/dev/null | grep -qE ':22([^0-9]|$)'; then
  echo "listen22=yes (key-only expected on day2 :52222 is primary)"
else
  echo "listen22=no"
fi
true`
	pout, perr := runSSHOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, probe22, o.PrimaryKeyPassphrase)
	if perr != nil {
		return fmt.Errorf("SSH harden end-state failed (PasswordAuthentication): %w\n%s", perr, pout)
	}
	fmt.Fprintln(os.Stderr, strings.TrimSpace(pout))
	_, _ = runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
		"ufw allow from 10.87.10.0/30 to any port 8789 proto tcp 2>/dev/null; ufw allow from 10.87.11.0/30 to any port 8789 proto tcp 2>/dev/null; ufw allow 8444/tcp 2>/dev/null; true",
		o.PrimaryKeyPassphrase)

	fmt.Fprintln(os.Stderr, "==> wait secondary online on primary (heartbeat)")
	if pack.AgentID == "" {
		return fmt.Errorf("prepare-pack missing agent_id — refuse to treat any online secondary as this deploy")
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
	out, err = runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, waitCmd, o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return fmt.Errorf("secondary not online on primary after deploy (agent heartbeat missing): %w\n%s", err, out)
	}

	// preferred_secondary_ip only when Promote (provision ≠ promotion).
	if !o.Promote {
		fmt.Fprintln(os.Stderr, "==> skip preferred_secondary_ip (Promote=false; use svc-paths peer promote)")
	}
	prefCmd := fmt.Sprintf(`id=%s
fallback=%s
line=$(netductor secondary status 2>/dev/null | awk -F '\t' -v id="$id" '$1 == id { print; exit }')
ip=$(printf '%%s\n' "$line" | awk -F'\t' '{for(i=1;i<=NF;i++) if($i ~ /^ip=/){sub(/^ip=/,"",$i); print $i; exit}}')
case "$ip" in
  ""|*[!0-9.]*)
    case "$fallback" in
      *[!0-9.]*) echo "preferred_secondary_ip: no IPv4 (skip)"; exit 0 ;;
      *) ip=$fallback ;;
    esac
    ;;
esac
mkdir -p /etc/netductor/secrets
printf '%%s\n' "$ip" > /etc/netductor/preferred_secondary_ip
printf '%%s\n' "$ip" > /etc/netductor/secrets/preferred_secondary_ip
chmod 600 /etc/netductor/preferred_secondary_ip /etc/netductor/secrets/preferred_secondary_ip 2>/dev/null
echo "preferred=$ip"
true`, ShellQuote(pack.AgentID), ShellQuote(o.SecondaryHost))
	if o.Promote {
		if outP, errP := runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, prefCmd, o.PrimaryKeyPassphrase); errP != nil {
			fmt.Fprintln(os.Stderr, "warn preferred_secondary_ip:", errP, outP)
		} else {
			fmt.Fprintln(os.Stderr, strings.TrimSpace(outP))
		}
	}

	fmt.Fprintln(os.Stderr, "==> secondary deploy done (Mac-direct; agent → primary :8789 mTLS via SP + svc-paths)")
	fmt.Fprintln(os.Stderr, "    note: public surface may include :443 (Reality), :52222 (SSH), :8445 (PS WSS)")
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
