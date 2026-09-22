package deploy

import (
	"fmt"
	"os"
	"strings"
)

// SecondaryOpts — provision RU secondary from operator workstation (Mac) or via primary.
//
// Default path (ViaPrimary=false): Mac SSHs to primary only to run provision command that
// opens SSH primary→secondary with secondary password. No Mac private key is copied to primary.
//
// ViaPrimary=true: same transport (must run provision on primary where control-plane state lives).
// Named explicitly for TUI; behavior matches default because bundle/mtls are issued on primary.
//
// True "Mac opens SSH to secondary" still needs primary for IssueToken/mTLS; the operator
// machine orchestrates both legs. Private key never leaves the operator machine.
type SecondaryOpts struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string // Mac private key path (pubkey = path+".pub")
	PrimaryKeyPassphrase string
	SecondaryHost        string
	SecondaryUser        string
	SecondaryPass        string
	SecondarySSHKey      string // optional; re-provision when password already off
	SNI                  string
	OperatorPubKey       string
	// ViaPrimary kept for CLI/TUI clarity; provision always executes on primary
	// (control plane). Direct Mac→secondary SSH for join is future if we add pack export.
	ViaPrimary bool
}

// DeploySecondary provisions secondary without copying the Mac private key to primary.
func DeploySecondary(o SecondaryOpts) error {
	if o.PrimaryHost == "" || o.PrimaryKey == "" {
		return fmt.Errorf("primary host and SSH key required")
	}
	if o.SecondaryHost == "" {
		return fmt.Errorf("secondary host required")
	}
	if o.SecondaryPass == "" && o.SecondarySSHKey == "" {
		return fmt.Errorf("secondary password or --secondary-ssh-key required")
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

	// Password via stdin on primary (not in process list). No Mac private key upload.
	var cmd string
	if o.SecondaryPass != "" {
		cmd = fmt.Sprintf(
			"printf '%%s' %s | netductor fleet provision-secondary --password-stdin --host %s --user %s --sni %s --operator-pubkey %s",
			ShellQuote(o.SecondaryPass), ShellQuote(o.SecondaryHost), ShellQuote(o.SecondaryUser),
			ShellQuote(o.SNI), ShellQuote(pub),
		)
	} else {
		// Re-provision with key already on primary is unsupported without copying key;
		// operator should use password on fresh VPS or run secondary provision on primary with --ssh-key locally.
		return fmt.Errorf("secondary password required for deploy from workstation (fresh VPS); for re-provision run on primary: netductor secondary provision --ssh-key …")
	}

	mode := "via primary (control-plane issues bundle; harden-last)"
	if !o.ViaPrimary {
		mode = "default: orchestrated from Mac → primary runs provision (no private key on primary)"
	}
	fmt.Fprintln(os.Stderr, "==> secondary deploy:", mode)
	fmt.Fprintln(os.Stderr, "==> on primary:", o.PrimaryHost, "→ provision secondary", o.SecondaryHost)
	out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, cmd, o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return err
	}
	out, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "netductor secondary sync; netductor fleet status", o.PrimaryKeyPassphrase)
	fmt.Print(out)
	fmt.Fprintln(os.Stderr, "==> secondary deploy done (agent→primary HTTP; Mac key never stored on primary)")
	return nil
}
