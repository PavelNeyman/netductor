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
	PrimaryHost    string
	PrimaryUser    string
	PrimaryKey     string
	PrimaryKeyPassphrase string
	RouterHost     string
	RouterUser     string
	RouterPass     string
	DeviceID       string
	ServerURL      string
	AgentArch      string
	Version        string
	AgentDir       string
	OperatorPubKey string
	GuestEnable    bool
	GuestSSID      string
	GuestPIN       string
	GuestPSK       string
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
	if o.AgentArch == "" {
		o.AgentArch = "arm64"
	}
	if !validReleaseVersion(o.AgentArch) { // same charset: arm64, armv7, amd64
		return fmt.Errorf("invalid agent arch %q", o.AgentArch)
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

	token := ""
	var mtlsCA, mtlsCert, mtlsKey []byte
	if o.PrimaryHost != "" && o.PrimaryKey != "" {
		out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
			"cat /etc/netductor/secrets/edge_bootstrap_token 2>/dev/null", o.PrimaryKeyPassphrase)
		if err != nil {
			return fmt.Errorf("read bootstrap token from primary: %w\n%s", err, out)
		}
		token = strings.TrimSpace(out)

		// Issue client cert on primary; print base64 lines (OpenWrt-safe path matching ClientDir).
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
		mout, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, remote, o.PrimaryKeyPassphrase)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: mtls material from primary:", err)
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
	
	if err := edge.Provision(edge.ProvisionOpts{
		SSHTarget:      target,
		DeviceID:       o.DeviceID,
		ServerURL:      o.ServerURL,
		AgentBin:       agent,
		Token:          token,
		Password:       o.RouterPass,
		SSHKey:         o.PrimaryKey,
		OperatorPubKey: pub,
		MTLSCA:         mtlsCA,
		MTLSCert:       mtlsCert,
		MTLSKey:        mtlsKey,
	}); err != nil {
		return err
	}
	if o.NetConfigure {
		if err := applyNetworkOnEdge(o); err != nil {
			fmt.Fprintln(os.Stderr, "warn: network apply:", err)
		}
	}
	if o.GuestEnable {
		if err := applyGuestOnEdge(o); err != nil {
			fmt.Fprintln(os.Stderr, "warn: guest enable:", err)
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
	script := edgeagent.ShellApply(desired)
	out, err := runSSH(o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, script, "")
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	fmt.Fprintln(os.Stderr, "==> network UCI applied on", o.RouterHost)
	return nil
}


func applyGuestOnEdge(o EdgeOpts) error {
	ssid := o.GuestSSID
	if ssid == "" {
		ssid = "Guest"
	}
	pin := o.GuestPIN
	psk := o.GuestPSK
	if o.PrimaryHost != "" {
		_ = edge.BindTemplate(o.DeviceID, "default", map[string]any{
			"guest": map[string]any{
				"enabled":  true,
				"ssid":     ssid,
				"desk_pin": pin,
				"psk":      psk,
				"hidden":   true,
			},
		})
		_ = edge.EnqueueCmd(o.DeviceID, "apply_template", "")
	}
	cmd := "netductor-agent guest enable --ssid=" + shellQuote(ssid) + " --pin=" + shellQuote(pin)
	if psk != "" {
		cmd += " --psk=" + shellQuote(psk)
	}
	out, err := runSSH(o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, cmd, "")
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
