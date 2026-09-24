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

// EdgeSpec — OpenWrt / RPi from operator machine.
type EdgeSpec struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string
	PrimaryKeyPassphrase string
	RouterHost           string
	RouterUser           string
	RouterPass           string
	DeviceID             string
	ServerURL            string
	Version              string
	AgentArch            string
	NetConfigure         bool
	LANIP, LANMask       string
	DHCPStart, DHCPLimit string
	WiFiSSID, WiFiKey    string
	WiFiSSID24, WiFiKey24 string
	WiFiSSID5, WiFiKey5  string
	GuestEnable          bool
	GuestSSID, GuestPIN, GuestPSK string
	WANProto, WANIP, WANMask, WANGateway, WANDNS string
	PPPoEUser, PPPoEPass, PPPoEService, PPPoEAC string
}

func (e EdgeSpec) toDeploy() deploy.EdgeOpts {
	return deploy.EdgeOpts{
		PrimaryHost: e.PrimaryHost, PrimaryUser: e.PrimaryUser, PrimaryKey: e.PrimaryKey,
		PrimaryKeyPassphrase: e.PrimaryKeyPassphrase,
		RouterHost: e.RouterHost, RouterUser: e.RouterUser, RouterPass: e.RouterPass,
		DeviceID: e.DeviceID, ServerURL: e.ServerURL, Version: e.Version, AgentArch: e.AgentArch,
		NetConfigure: e.NetConfigure, LANIP: e.LANIP, LANMask: e.LANMask,
		DHCPStart: e.DHCPStart, DHCPLimit: e.DHCPLimit,
		WiFiSSID: e.WiFiSSID, WiFiKey: e.WiFiKey,
		WiFiSSID24: e.WiFiSSID24, WiFiKey24: e.WiFiKey24,
		WiFiSSID5: e.WiFiSSID5, WiFiKey5: e.WiFiKey5,
		GuestEnable: e.GuestEnable, GuestSSID: e.GuestSSID, GuestPIN: e.GuestPIN, GuestPSK: e.GuestPSK,
		WANProto: e.WANProto, WANIP: e.WANIP, WANMask: e.WANMask, WANGateway: e.WANGateway, WANDNS: e.WANDNS,
		PPPoEUser: e.PPPoEUser, PPPoEPass: e.PPPoEPass, PPPoEService: e.PPPoEService, PPPoEAC: e.PPPoEAC,
	}
}
