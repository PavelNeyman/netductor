package secondary

import (
	"fmt"
	"strings"
)

// SSHTransportMode is the authentication mode of the active provision transport.
type SSHTransportMode int

const (
	// SSHTransportBootstrap allows password (first contact only).
	SSHTransportBootstrap SSHTransportMode = iota
	// SSHTransportKeyVerified: password must not be used; key-only session verified.
	SSHTransportKeyVerified
	// SSHTransportHardened: key-only on DefaultSSHPort after finalize.
	SSHTransportHardened
)

func (m SSHTransportMode) String() string {
	switch m {
	case SSHTransportBootstrap:
		return "bootstrap"
	case SSHTransportKeyVerified:
		return "key_verified"
	case SSHTransportHardened:
		return "hardened"
	default:
		return fmt.Sprintf("mode(%d)", int(m))
	}
}

// SSHTransport is the single source of truth for Secondary provision SSH auth mode.
// BootstrapCredentials live only until KeyVerified; they are never persisted.
type SSHTransport struct {
	Mode SSHTransportMode
	Host string
	Port int
	User string

	KeyPath       string
	KeyPassphrase string
	bootstrapPass string // zeroed on PromoteToKeyVerified
}

// NewSSHTransportBootstrap builds bootstrap transport. Password is held only in memory.
func NewSSHTransportBootstrap(host string, port int, user, password, keyPath, keyPass string) *SSHTransport {
	if port <= 0 {
		port = 22
	}
	if user == "" {
		user = "root"
	}
	return &SSHTransport{
		Mode:          SSHTransportBootstrap,
		Host:          host,
		Port:          port,
		User:          user,
		KeyPath:       strings.TrimSpace(keyPath),
		KeyPassphrase: keyPass,
		bootstrapPass: password,
	}
}

// BootstrapPassword returns the password only while Mode == Bootstrap.
func (t *SSHTransport) BootstrapPassword() (string, error) {
	if t == nil {
		return "", fmt.Errorf("ssh transport: nil")
	}
	if t.Mode != SSHTransportBootstrap {
		return "", fmt.Errorf("ssh transport: password unavailable in mode %s", t.Mode)
	}
	return t.bootstrapPass, nil
}

// PromoteToKeyVerified clears bootstrap password and switches mode.
func (t *SSHTransport) PromoteToKeyVerified() error {
	if t == nil {
		return fmt.Errorf("ssh transport: nil")
	}
	if t.Mode != SSHTransportBootstrap {
		return fmt.Errorf("ssh transport: cannot promote from %s", t.Mode)
	}
	if strings.TrimSpace(t.KeyPath) == "" {
		return fmt.Errorf("ssh transport: key path required for key_verified")
	}
	t.bootstrapPass = ""
	t.Mode = SSHTransportKeyVerified
	return nil
}

// PromoteToHardened after successful finalize + post key-only check on hardened port.
func (t *SSHTransport) PromoteToHardened(port int) error {
	if t == nil {
		return fmt.Errorf("ssh transport: nil")
	}
	if t.Mode != SSHTransportKeyVerified {
		return fmt.Errorf("ssh transport: cannot harden from %s", t.Mode)
	}
	if port > 0 {
		t.Port = port
	}
	t.Mode = SSHTransportHardened
	t.bootstrapPass = ""
	return nil
}

// Provision stages (error / log identifiers).
const (
	StageBootstrapConnect    = "secondary-bootstrap-connect"
	StageInstallPublicKey    = "secondary-install-public-key"
	StageKeyOnlyVerification = "secondary-key-only-verification"
	StageTransportSwitch     = "secondary-transport-switch"
	StageInstallComponents   = "secondary-install-components"
	StageConfigureServices   = "secondary-configure-services"
	StageSSHHardenPrepare    = "secondary-ssh-hardening-prepare"
	StageSSHHardenApply      = "secondary-ssh-hardening-apply"
	StageNewPortVerification = "secondary-new-port-verification"
	StagePasswordAuthDisable = "secondary-password-auth-disable"
	StageHardenFinalize      = "secondary-hardening-finalize"
)
