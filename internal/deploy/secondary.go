package deploy

import (
	"fmt"
	"os"
)

// SecondaryOpts — provision RU secondary by driving primary over SSH.
type SecondaryOpts struct {
	PrimaryHost   string
	PrimaryUser   string
	PrimaryKey    string
	SecondaryHost string
	SecondaryUser string
	SecondaryPass string
	SNI           string
}

// DeploySecondary runs `netductor fleet provision-secondary` on primary.
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
	cmd := fmt.Sprintf("netductor fleet provision-secondary --host %s --user %s --password %s --sni %s",
		shellQuote(o.SecondaryHost), shellQuote(o.SecondaryUser), shellQuote(o.SecondaryPass), shellQuote(o.SNI))
	fmt.Fprintln(os.Stderr, "==> on primary:", o.PrimaryHost, "\u2192 provision secondary", o.SecondaryHost)
	out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, cmd)
	fmt.Print(out)
	if err != nil {
		return err
	}
	out, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "netductor secondary sync; netductor fleet status")
	fmt.Print(out)
	return nil
}
