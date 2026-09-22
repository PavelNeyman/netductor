package deploy

import (
	"fmt"
	"os"
	"strings"
)

// SecondaryOpts — provision RU secondary by driving primary over SSH.
// Operator (Mac) pubkey is installed on secondary; ongoing ops use agent→primary HTTP.
type SecondaryOpts struct {
	PrimaryHost     string
	PrimaryUser     string
	PrimaryKey      string // path to Mac private key (pubkey = path+".pub")
	SecondaryHost   string
	SecondaryUser   string
	SecondaryPass   string
	SNI             string
	OperatorPubKey  string // optional explicit pubkey line; else read PrimaryKey+".pub"
}

// DeploySecondary runs `netductor fleet provision-secondary` on primary with --operator-pubkey.
func DeploySecondary(o SecondaryOpts) error {
	if o.PrimaryHost == "" || o.PrimaryKey == "" {
		return fmt.Errorf("primary host and SSH key required")
	}
	if o.SecondaryHost == "" || o.SecondaryPass == "" {
		return fmt.Errorf("secondary host and password required")
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
	// Password via stdin on primary (not netductor CLI argv / process list).
	cmd := fmt.Sprintf(
		"printf '%%s' %s | netductor fleet provision-secondary --password-stdin --host %s --user %s --sni %s --operator-pubkey %s",
		ShellQuote(o.SecondaryPass), ShellQuote(o.SecondaryHost), ShellQuote(o.SecondaryUser),
		ShellQuote(o.SNI), ShellQuote(pub),
	)
	fmt.Fprintln(os.Stderr, "==> on primary:", o.PrimaryHost, "→ provision secondary", o.SecondaryHost, "(operator pubkey)")
	out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, cmd)
	fmt.Print(out)
	if err != nil {
		return err
	}
	out, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "netductor secondary sync; netductor fleet status")
	fmt.Print(out)
	return nil
}
