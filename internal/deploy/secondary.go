package deploy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
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
	if os.Getenv("NETDUCTOR_SSH_PORT") == "" {
		_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	}
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
		o.SNI = "api.vk.me"
	}
	pub := strings.TrimSpace(o.OperatorPubKey)
	if pub == "" {
		b, err := os.ReadFile(o.PrimaryKey + ".pub")
		if err != nil {
			return fmt.Errorf("read operator pubkey %s.pub: %w", o.PrimaryKey, err)
		}
		pub = strings.TrimSpace(string(b))
	}
	if pub == "" {
		return fmt.Errorf("operator pubkey empty")
	}

	fmt.Fprintln(os.Stderr, "==> secondary deploy: Mac-direct (primary prepares pack; Mac SSHs to secondary)")
	fmt.Fprintln(os.Stderr, "==> 1/2 primary prepare-pack on", o.PrimaryHost)

	prep := "netductor secondary prepare-pack --sni " + ShellQuote(o.SNI)
	out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, prep, o.PrimaryKeyPassphrase)
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
	// Secondary first login is :22; do not use primary's 52222 env for this leg.
	prevPort := os.Getenv("NETDUCTOR_SSH_PORT")
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "22")
	defer func() {
		if prevPort != "" {
			_ = os.Setenv("NETDUCTOR_SSH_PORT", prevPort)
		} else {
			_ = os.Unsetenv("NETDUCTOR_SSH_PORT")
		}
	}()

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

	// post: sync/fleet on primary
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	fmt.Fprintln(os.Stderr, "==> primary: secondary sync + fleet status")
	out, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
		"netductor secondary sync; netductor fleet bootstrap 2>/dev/null; netductor fleet status",
		o.PrimaryKeyPassphrase)
	fmt.Print(out)

	// credentials: after harden SSH is on :52222 (same as primary)
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	if path, err := CollectOperatorSecrets("secondary", o.SecondaryUser, o.SecondaryHost, o.PrimaryKey, o.PrimaryKeyPassphrase); err != nil {
		fmt.Fprintln(os.Stderr, "warn: credentials collect:", err)
	} else {
		fmt.Fprintln(os.Stderr, "  Credentials file:", path)
	}
	fmt.Fprintln(os.Stderr, "==> secondary deploy done (Mac-direct; agent → primary :8789 mTLS)")
	return nil
}
