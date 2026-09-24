// Package operator is the Mac/workstation use-case layer.
// CLI and TUI must call these functions only — not assemble deploy logic themselves.
package operator

import (
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

// PrimarySpec is the deploy-primary use-case input (shared by CLI/TUI/Web).
type PrimarySpec struct {
	Host            string
	User            string
	Password        string
	SSHPrivateKey   string
	GenerateKey     bool
	KeyPassphrase   string
	Version         string
	TelegramToken   string
	TelegramAdminID string
	SNI             string
	DomainBase      string
	DomainHTTP      bool
	DomainLE        bool
	DomainCFProxy   bool
	DomainEmail     string
	SkipInstall     bool
	WithLampac      bool
	WithGitRegistry bool
}

// SecondarySpec is the deploy-secondary use-case input.
type SecondarySpec struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string
	PrimaryKeyPassphrase string
	SecondaryHost        string
	SecondaryUser        string
	SecondaryPass        string
	SecondarySSHKey      string
	SNI                  string
	OperatorPubKey       string
}

// FleetSpec runs ordered primary then secondary (Mac operator).
type FleetSpec struct {
	DoPrimary   bool
	DoSecondary bool
	Primary     PrimarySpec
	Secondary   SecondarySpec
}

func (p PrimarySpec) toDeploy() deploy.PrimaryOpts {
	return deploy.PrimaryOpts{
		Host: p.Host, User: p.User, Password: p.Password,
		SSHPrivateKey: p.SSHPrivateKey, GenerateKey: p.GenerateKey, KeyPassphrase: p.KeyPassphrase,
		Version: p.Version, TelegramToken: p.TelegramToken, TelegramAdminID: p.TelegramAdminID,
		SNI: p.SNI, DomainBase: p.DomainBase, DomainHTTP: p.DomainHTTP, DomainLE: p.DomainLE,
		DomainCFProxy: p.DomainCFProxy, DomainEmail: p.DomainEmail, SkipInstall: p.SkipInstall,
		WithLampac: p.WithLampac, WithGitRegistry: p.WithGitRegistry,
	}
}

func (s SecondarySpec) toDeploy() deploy.SecondaryOpts {
	return deploy.SecondaryOpts{
		PrimaryHost: s.PrimaryHost, PrimaryUser: s.PrimaryUser, PrimaryKey: s.PrimaryKey,
		PrimaryKeyPassphrase: s.PrimaryKeyPassphrase,
		SecondaryHost: s.SecondaryHost, SecondaryUser: s.SecondaryUser,
		SecondaryPass: s.SecondaryPass, SecondarySSHKey: s.SecondarySSHKey,
		SNI: s.SNI, OperatorPubKey: s.OperatorPubKey, ViaPrimary: false,
	}
}

// ApplyDomainFlags sets DomainLE/HTTP consistently from base + email.
func ApplyDomainFlags(p *PrimarySpec) {
	base := strings.TrimSpace(p.DomainBase)
	email := strings.TrimSpace(p.DomainEmail)
	if base == "" {
		p.DomainLE, p.DomainHTTP = false, false
		return
	}
	if email != "" {
		p.DomainLE = true
		p.DomainHTTP = false
		p.DomainEmail = email
		return
	}
	p.DomainLE = false
	p.DomainHTTP = true
}
