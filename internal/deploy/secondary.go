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
	PrimaryKeyPassphrase string
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
	// Leave operator private key on primary for post-provision scp (mtls seed, offsite backup).
	// Password auth on secondary is disabled after first provision.
	fmt.Fprintln(os.Stderr, "==> install operator key on primary for secondary post-steps")
	pubPath := o.PrimaryKey + ".pub"
	installKeyCmd := fmt.Sprintf(
		"install -m 600 /dev/null /root/.ssh/operator_reprovision 2>/dev/null; true")
	_, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, installKeyCmd, o.PrimaryKeyPassphrase)
	_ = scpTo(o.PrimaryKey, o.PrimaryKey, o.PrimaryUser+"@"+o.PrimaryHost+":/root/.ssh/operator_reprovision", o.PrimaryKeyPassphrase)
	_ = scpTo(o.PrimaryKey, pubPath, o.PrimaryUser+"@"+o.PrimaryHost+":/root/.ssh/operator_reprovision.pub", o.PrimaryKeyPassphrase)
	_, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "chmod 600 /root/.ssh/operator_reprovision", o.PrimaryKeyPassphrase)

	fmt.Fprintln(os.Stderr, "==> on primary:", o.PrimaryHost, "→ provision secondary", o.SecondaryHost, "(operator pubkey)")
	out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, cmd, o.PrimaryKeyPassphrase)
	fmt.Print(out)
	if err != nil {
		return err
	}
	out, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "netductor secondary sync; netductor fleet status", o.PrimaryKeyPassphrase)
	fmt.Print(out)
	return nil
}
