package deploy

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/edgeagent"
	"github.com/PavelNeyman/netductor/internal/mtls"
)

type EdgeOpts struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string
	PrimaryKeyPassphrase string
	// Offline: skip SSH to primary when BootstrapToken is set (optional local mTLS files).
	BootstrapToken string
	MTLSCAFile     string
	MTLSCertFile   string
	MTLSKeyFile    string
	RouterHost           string
	RouterUser           string
	RouterPass           string
	NewRootPassword      string // LuCI/root pass; required unless SkipRootPass
	SkipRootPass         bool
	DeviceID             string
	ServerURL            string
	AgentArch            string
	Version              string
	AgentDir             string
	OperatorPubKey       string
	GuestEnable          bool
	GuestSSID            string
	GuestPIN             string
	GuestPSK             string
	GuestVisible         bool // true = broadcast SSID (default for forms); false = hidden
	// Optional first-boot network (applied via UCI on router + edge template)
	NetConfigure bool
	LANIP        string
	LANMask      string
	DHCPStart    string // e.g. 100
	DHCPLimit    string // e.g. 150
	WiFiSSID     string // legacy both bands
	WiFiKey      string
	WiFiSSID24   string
	WiFiKey24    string
	WiFiSSID5    string
	WiFiKey5     string
	WANProto     string // dhcp | static | pppoe
	WANIP        string
	WANMask      string
	WANGateway   string
	WANDNS       string
	PPPoEUser    string
	PPPoEPass    string
	PPPoEService string
	PPPoEAC      string
	// Reboot router after provision (agent init + optional network).
	Reboot bool
}

func DeployEdge(o EdgeOpts) error {
	if o.RouterHost == "" || o.DeviceID == "" {
		return fmt.Errorf("router host and device id required")
	}
	if o.RouterUser == "" {
		o.RouterUser = "root"
	}
	if o.PrimaryUser == "" {
		o.PrimaryUser = "root"
	}
	if o.Version == "" {
		o.Version = Release
	}
	if !validReleaseVersion(o.Version) {
		return fmt.Errorf("invalid release version %q", o.Version)
	}
	if !ValidAgentArch(o.AgentArch) {
		return fmt.Errorf("invalid agent arch %q (amd64|arm64|arm|mipsle|riscv64|auto)", o.AgentArch)
	}
	if o.AgentDir == "" {
		home, _ := os.UserHomeDir()
		o.AgentDir = filepath.Join(home, ".cache", "netductor", "agents")
	}
	if o.ServerURL == "" && o.PrimaryHost != "" {
		o.ServerURL = "https://" + o.PrimaryHost + ":" + mtls.AgentTLSPort
	}
	if o.ServerURL == "" {
		return fmt.Errorf("server URL or primary host required")
	}
	// Never use plain public admin :8787 for edge control.
	if strings.HasPrefix(o.ServerURL, "http://") {
		u := strings.TrimPrefix(o.ServerURL, "http://")
		u = strings.Replace(u, ":8787", ":"+mtls.AgentTLSPort, 1)
		if !strings.Contains(u, ":") {
			u = u + ":" + mtls.AgentTLSPort
		}
		o.ServerURL = "https://" + u
	}
	if !strings.HasPrefix(o.ServerURL, "https://") {
		return fmt.Errorf("server URL must be https:// (agent plane)")
	}
	if serverURLUnsafe(o.ServerURL) {
		return fmt.Errorf("invalid server URL characters")
	}

	token := strings.TrimSpace(o.BootstrapToken)
	var mtlsCA, mtlsCert, mtlsKey []byte
	// Local mTLS material (offline / pre-copied from primary).
	if o.MTLSCAFile != "" {
		if b, err := os.ReadFile(o.MTLSCAFile); err == nil {
			mtlsCA = b
		}
	}
	if o.MTLSCertFile != "" {
		if b, err := os.ReadFile(o.MTLSCertFile); err == nil {
			mtlsCert = b
		}
	}
	if o.MTLSKeyFile != "" {
		if b, err := os.ReadFile(o.MTLSKeyFile); err == nil {
			mtlsKey = b
		}
	}
	// Online: pull bootstrap token from primary over SSH :52222.
	if token == "" && o.PrimaryHost != "" && o.PrimaryKey != "" {
		out, err := runSSHOnPort(day2SSHPort(), "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
			"cat /etc/netductor/secrets/edge_bootstrap_token 2>/dev/null", o.PrimaryKeyPassphrase)
		if err != nil {
			return fmt.Errorf("read bootstrap token from primary: %w\n%s\n(hint: offline — pass BootstrapToken from a prior pull)", err, out)
		}
		token = strings.TrimSpace(out)
	}
	if token == "" {
		return fmt.Errorf("edge bootstrap token required (SSH to primary or BootstrapToken / --bootstrap-token)")
	}
	// Online: issue client cert on primary unless local mTLS already loaded.
	if len(mtlsCert) == 0 && o.PrimaryHost != "" && o.PrimaryKey != "" {
		remote := fmt.Sprintf(`set -e
netductor mtls ensure >/dev/null 2>&1 || true
netductor mtls issue-client %s >/dev/null 2>&1
CA=/etc/netductor/secrets/mtls/ca.crt
DIR=/etc/netductor/secrets/mtls/clients/%s
if [ ! -f "$DIR/client.crt" ]; then
  DIR=$(find /etc/netductor/secrets/mtls/clients -maxdepth 1 -type d 2>/dev/null | while read d; do
    [ -f "$d/client.crt" ] || continue
    echo "$d"
  done | head -1)
fi
b64() { base64 -w0 "$1" 2>/dev/null || base64 "$1" | tr -d '\n'; }
echo CA:$(b64 "$CA")
echo CERT:$(b64 "$DIR/client.crt")
echo KEY:$(b64 "$DIR/client.key")
`, shellQuote(o.DeviceID), shellQuote(o.DeviceID))
		mout, err := runSSHOnPort(day2SSHPort(), "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, remote, o.PrimaryKeyPassphrase)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: mTLS material from primary:", err)
			fmt.Fprintln(os.Stderr, mout)
		} else {
			for _, line := range strings.Split(mout, "\n") {
				line = strings.TrimSpace(line)
				switch {
				case strings.HasPrefix(line, "CA:"):
					mtlsCA, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "CA:"))
				case strings.HasPrefix(line, "CERT:"):
					mtlsCert, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "CERT:"))
				case strings.HasPrefix(line, "KEY:"):
					mtlsKey, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "KEY:"))
				}
			}
		}
	}

	// SSH probe first (unless --arch override); Cudy TR1200 = mipsle, not arm64.
	arch, err := ResolveAgentArch(o.AgentArch, o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, "")
	if err != nil {
		return err
	}
	o.AgentArch = arch

	agent, err := EnsureAgentBinary(o.Version, o.AgentArch, o.AgentDir)
	if err != nil {
		return err
	}
	pub := strings.TrimSpace(o.OperatorPubKey)
	if pub == "" && o.PrimaryKey != "" {
		if b, err := os.ReadFile(o.PrimaryKey + ".pub"); err == nil {
			pub = strings.TrimSpace(string(b))
		}
	}
	target := o.RouterUser + "@" + o.RouterHost
	fmt.Fprintln(os.Stderr, "==> edge provision", target, "→", o.ServerURL)
	if len(mtlsCert) > 0 {
		fmt.Fprintln(os.Stderr, "==> mTLS client material will be installed on router")
	} else {
		fmt.Fprintln(os.Stderr, "warn: no mTLS material — agent may fail TLS handshake until certs are present")
	}

	// 1) Agent + config (+ optional new root pass) WITHOUT harden — password still works for network/guest.
	needNet := o.NetConfigure || o.GuestEnable
	if err := edge.Provision(edge.ProvisionOpts{
		SSHTarget:       target,
		DeviceID:        o.DeviceID,
		ServerURL:       o.ServerURL,
		AgentBin:        agent,
		Token:           token,
		Password:        o.RouterPass,
		NewRootPassword: o.NewRootPassword,
		SSHKey:          o.PrimaryKey,
		OperatorPubKey:  pub,
		MTLSCA:          mtlsCA,
		MTLSCert:        mtlsCert,
		MTLSKey:         mtlsKey,
		SkipHarden:      true, // harden after network/guest
	}); err != nil {
		return err
	}
	// After setPass, subsequent SSH must use the NEW root password.
	sshPass := o.RouterPass
	if np := strings.TrimSpace(o.NewRootPassword); np != "" {
		sshPass = np
	}
	o.RouterPass = sshPass

	// 2) Stage network/guest while password auth still enabled.
	if needNet {
		o.Reboot = true
		fmt.Fprintln(os.Stderr, "==> network/guest changes staged; reboot forced at end")
	}
	if o.NetConfigure {
		if err := applyNetworkOnEdge(o); err != nil {
			fmt.Fprintln(os.Stderr, "warn: network stage:", err)
		}
	}
	if o.GuestEnable {
		if err := applyGuestOnEdge(o); err != nil {
			fmt.Fprintln(os.Stderr, "warn: guest stage:", err)
		}
	}

	// 3) Harden (pubkey + disable password) then reboot.
	if strings.TrimSpace(pub) != "" {
		fmt.Fprintln(os.Stderr, "==> harden SSH (pubkey, disable password)")
		if err := edge.Harden(edge.ProvisionOpts{
			SSHTarget: target, Password: sshPass, SSHKey: o.PrimaryKey, OperatorPubKey: pub,
		}); err != nil {
			fmt.Fprintln(os.Stderr, "warn: harden:", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "warn: no operator pubkey — skip harden (password SSH remains)")
	}

	if o.Reboot {
		fmt.Fprintln(os.Stderr, "==> reboot", o.RouterHost)
		if o.NetConfigure && o.LANIP != "" {
			fmt.Fprintln(os.Stderr, "==> after reboot SSH may move to", o.LANIP, "(staged LAN)")
		}
		// Prefer key after harden; password may already be off.
		out, err := runSSHOnPort(factorySSHPort(), "", o.PrimaryKey, o.RouterUser, o.RouterHost, "sync; reboot", "")
		if err != nil && sshPass != "" {
			out, err = runSSHOnPort(factorySSHPort(), sshPass, o.PrimaryKey, o.RouterUser, o.RouterHost, "sync; reboot", "")
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: reboot:", err, out)
		} else {
			fmt.Fprintln(os.Stderr, "==> reboot issued; agent enrolls after boot; approve on primary")
		}
	}
	return nil
}

func networkTemplateMap(o EdgeOpts) map[string]any {
	m := map[string]any{}
	net := map[string]any{}
	if o.LANIP != "" {
		net["lan_ip"] = o.LANIP
	}
	if o.LANMask != "" {
		net["lan_mask"] = o.LANMask
	} else if o.LANIP != "" {
		net["lan_mask"] = "255.255.255.0"
	}
	proto := strings.ToLower(strings.TrimSpace(o.WANProto))
	if proto == "" {
		proto = "dhcp"
	}
	net["wan_proto"] = proto
	if proto == "static" {
		net["wan_ip"] = o.WANIP
		net["wan_mask"] = o.WANMask
		if o.WANMask == "" {
			net["wan_mask"] = "255.255.255.0"
		}
		net["wan_gateway"] = o.WANGateway
		net["wan_dns"] = o.WANDNS
	}
	if proto == "pppoe" {
		net["pppoe_user"] = o.PPPoEUser
		net["pppoe_pass"] = o.PPPoEPass
		net["pppoe_service"] = o.PPPoEService
		net["pppoe_ac"] = o.PPPoEAC
		if o.WANDNS != "" {
			net["wan_dns"] = o.WANDNS
		}
	}
	if len(net) > 0 {
		m["network"] = net
	}
	if o.DHCPStart != "" || o.DHCPLimit != "" {
		m["dhcp"] = map[string]any{"start": o.DHCPStart, "limit": o.DHCPLimit}
	}
	wifi := map[string]any{"encryption": "psk2"}
	if o.WiFiSSID24 != "" || o.WiFiKey24 != "" || o.WiFiSSID5 != "" || o.WiFiKey5 != "" {
		wifi["ssid_24"] = o.WiFiSSID24
		wifi["key_24"] = o.WiFiKey24
		wifi["ssid_5"] = o.WiFiSSID5
		wifi["key_5"] = o.WiFiKey5
	} else if o.WiFiSSID != "" {
		wifi["ssid"] = o.WiFiSSID
		wifi["key"] = o.WiFiKey
	}
	if o.WiFiSSID24 != "" || o.WiFiSSID5 != "" || o.WiFiSSID != "" {
		m["wifi"] = wifi
	}
	return m
}

func applyNetworkOnEdge(o EdgeOpts) error {
	tmpl := networkTemplateMap(o)
	if len(tmpl) == 0 {
		return nil
	}
	if o.PrimaryHost != "" && o.DeviceID != "" {
		_ = edge.BindTemplate(o.DeviceID, "default", tmpl)
		_ = edge.EnqueueCmd(o.DeviceID, "apply_template", "")
	}
	desired := edgeagent.DesiredUCI(tmpl)
	script := edgeagent.ShellApplyStaged(desired)
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, script, "")
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	fmt.Fprintln(os.Stderr, "==> network UCI staged (commit, no reload) on", o.RouterHost)
	return nil
}

func applyGuestOnEdge(o EdgeOpts) error {
	ssid := o.GuestSSID
	if ssid == "" {
		ssid = "Guest"
	}
	pin := o.GuestPIN
	psk := o.GuestPSK
	// Persist guest into edge template only when we have a device id (day-2 apply_template).
	// First-boot source of truth is SSH guest enable below.
	if o.PrimaryHost != "" && strings.TrimSpace(o.DeviceID) != "" {
		_ = edge.BindTemplate(o.DeviceID, "default", map[string]any{
			"guest": map[string]any{
				"enabled":  true,
				"ssid":     ssid,
				"desk_pin": pin,
				"psk":      psk,
				"hidden":   !o.GuestVisible,
			},
		})
		_ = edge.EnqueueCmd(o.DeviceID, "apply_template", "")
	}
	cmd := "netductor-agent guest enable --ssid=" + shellQuote(ssid) + " --pin=" + shellQuote(pin)
	if psk != "" {
		cmd += " --psk=" + shellQuote(psk)
	}
	if o.GuestVisible {
		cmd += " --hidden=0"
	} else {
		cmd += " --hidden=1"
	}
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, cmd, "")
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	fmt.Fprintln(os.Stderr, "==> guest Wi-Fi enable on", o.RouterHost)
	return nil
}

func serverURLUnsafe(u string) bool {
	for _, r := range u {
		switch r {
		case ' ', '\t', '\n', '\r', ';', '|', '&', '$', '`', '"', '\'', '\\', '<', '>', '(', ')', '{', '}':
			return true
		}
	}
	return false
}
