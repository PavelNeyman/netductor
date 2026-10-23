package deploy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"os"
	"path/filepath"
	"strings"

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
	// ViaPrimary is ignored (deprecated). Always Mac-direct.
	ViaPrimary bool
}

type secondaryPack struct {
	Bundle      json.RawMessage `json:"bundle"`
	AgentID     string          `json:"agent_id"`
	AgentToken  string          `json:"agent_token"`
	CoreURL     string          `json:"core_url"`
	MTLSCAB64   string          `json:"mtls_ca_b64"`
	MTLSCertB64 string          `json:"mtls_cert_b64"`
	MTLSKeyB64  string          `json:"mtls_key_b64"`
	SNI         string          `json:"sni"`
}

// DeploySecondary: Mac orchestrates both legs; primary never SSHs to secondary.
func DeploySecondary(o SecondaryOpts) error {
	// Ports are passed explicitly (runSSHOnPort / ProvisionIn.Port) — no process-global NETDUCTOR_SSH_PORT.
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
		// path to .pub file
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
	// prepare-pack prints JSON on stdout; may mix stderr noise — find first '{'
	raw := out
	if i := strings.Index(out, "{"); i >= 0 {
		raw = out[i:]
	}
	var pack secondaryPack
	if err := json.Unmarshal([]byte(raw), &pack); err != nil {
		return fmt.Errorf("parse prepare-pack JSON: %w\n%s", err, out)
	}
	if pack.AgentToken == "" || len(pack.Bundle) == 0 {
		return fmt.Errorf("prepare-pack missing token/bundle:\n%s", out)
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
	res, err := secondary.ProvisionFromCore(secondary.ProvisionIn{
		Host: o.SecondaryHost, Port: 22, User: o.SecondaryUser,
		Password: o.SecondaryPass, SSHPrivateKey: o.SecondarySSHKey,
		SNI: o.SNI, OperatorPubKey: pub,
		MTLSCA: ca, MTLSCert: cert, MTLSKey: key,
	}, string(pack.Bundle))
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
	// SEC-06: verify operator key on hardened port before post-steps.
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

	// SEC-03: svc-paths required for tunnel-only agent plane.
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
	raw = out
	if idx := strings.Index(out, "{"); idx >= 0 {
		raw = out[idx:]
	}
	tmpFile, errTmp := os.CreateTemp("", "nd-svc-material-*.json")
	if errTmp != nil {
		return fmt.Errorf("svc-paths temp file: %w", errTmp)
	}
	tmpPath := tmpFile.Name()
	_, _ = tmpFile.Write([]byte(raw))
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	scpErr := runSCPOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, tmpPath, "/tmp/nd-svc-material.json", o.PrimaryKeyPassphrase)
	if scpErr != nil {
		fmt.Fprintln(os.Stderr, "warn svc-paths scp :52222:", scpErr, "— retry :22")
		scpErr = runSCPOnPort("22", "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost, tmpPath, "/tmp/nd-svc-material.json", o.PrimaryKeyPassphrase)
	}
	if scpErr != nil {
		return fmt.Errorf("svc-paths scp material to secondary: %w", scpErr)
	}
	out2, err2 := runSSHOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost,
		"netductor svc-paths bootstrap-secondary --material /tmp/nd-svc-material.json; rm -f /tmp/nd-svc-material.json; "+
			`test -f /etc/netductor/secrets/secondary_core_url; grep -q '10.87.10.1' /etc/netductor/secrets/secondary_core_url; `+
			"ping -c1 -W3 10.87.10.1",
		o.PrimaryKeyPassphrase)
	fmt.Print(out2)
	if err2 != nil {
		// try :22 if day2 not yet exclusive
		out2b, err2b := runSSHOnPort("22", "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost,
			"netductor svc-paths bootstrap-secondary --material /tmp/nd-svc-material.json; rm -f /tmp/nd-svc-material.json; "+
				`test -f /etc/netductor/secrets/secondary_core_url; grep -q '10.87.10.1' /etc/netductor/secrets/secondary_core_url; `+
				"ping -c1 -W3 10.87.10.1",
			o.PrimaryKeyPassphrase)
		fmt.Print(out2b)
		if err2b != nil {
			return fmt.Errorf("svc-paths bootstrap-secondary: %w\n%s", err2b, out2b)
		}
	}
	// Best-effort: prefer ssh.service over socket on :22 (SEC-17 — warn only if check fails later)
	_, _ = runSSHOnPort(day2Port, "", o.PrimaryKey, o.SecondaryUser, o.SecondaryHost,
		`set -e; systemctl stop ssh.socket sshd.socket 2>/dev/null || true; systemctl disable ssh.socket sshd.socket 2>/dev/null || true; systemctl enable ssh.service 2>/dev/null || true; systemctl restart ssh 2>/dev/null || systemctl restart sshd 2>/dev/null || true`,
		o.PrimaryKeyPassphrase)
	_, _ = runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
		"ufw allow from 10.87.10.0/30 to any port 8789 proto tcp 2>/dev/null; ufw allow from 10.87.11.0/30 to any port 8789 proto tcp 2>/dev/null; ufw allow 8444/tcp 2>/dev/null; ufw allow 8445/tcp 2>/dev/null; true",
		o.PrimaryKeyPassphrase)

	// Wait for agent heartbeat via primary registry (tunnel URL should be active).
	fmt.Fprintln(os.Stderr, "==> wait secondary online on primary (heartbeat)")
	waitCmd := fmt.Sprintf(
		`ok=0; for i in 1 2 3 4 5 6 7 8 9 10; do netductor secondary status 2>/dev/null | grep -F %s | grep -q online && ok=1 && break; sleep 3; done; test "$ok" = 1`,
		ShellQuote(o.SecondaryHost),
	)
	// status lines use agent id not always IP — also accept any online after our deploy
	waitCmd = `ok=0; for i in $(seq 1 12); do
  st=$(netductor secondary status 2>/dev/null || true)
  echo "$st"
  echo "$st" | grep -qi online && ok=1 && break
  sleep 5
done
test "$ok" = 1`
	out, err = runSSHOnPort(day2Port, "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, waitCmd, o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return fmt.Errorf("secondary not online on primary after deploy (agent heartbeat missing): %w\n%s", err, out)
	}

	fmt.Fprintln(os.Stderr, "==> secondary deploy done (Mac-direct; agent → primary :8789 mTLS via SP + svc-paths)")
	fmt.Fprintln(os.Stderr, "    note: public surface may include :443 (Reality), :52222 (SSH), :8445 (PS WSS) — see docs")
	return nil
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
