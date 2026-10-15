package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edgeagent"
)

// applyVPNClientOnEdge installs sing-box on the router and, when primary is reachable,
// creates the edge VPN user and writes client config (max work before agent→primary enroll).
func applyVPNClientOnEdge(o EdgeOpts) error {
	arch := o.AgentArch
	if arch == "" || arch == "auto" {
		return fmt.Errorf("vpn_client: agent arch required (probe first)")
	}
	fmt.Fprintln(os.Stderr, "==> vpn_client: install sing-box binary for", arch)
	localBin, err := EnsureSingBoxCached(arch)
	if err != nil {
		return fmt.Errorf("sing-box cache: %w", err)
	}
	pass := o.RouterPass
	user := o.RouterUser
	host := o.RouterHost
	port := factorySSHPort()
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, localBin, "/usr/sbin/sing-box", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp sing-box: %w", err)
	}
	if _, err := runSSHOnPort(port, pass, o.PrimaryKey, user, host, "chmod 755 /usr/sbin/sing-box", o.PrimaryKeyPassphrase); err != nil {
		fmt.Fprintln(os.Stderr, "warn: chmod sing-box:", err)
	}

	vless := ""
	if o.PrimaryHost != "" && o.PrimaryKey != "" {
		fmt.Fprintln(os.Stderr, "==> vpn_client: ensure edge user on primary")
		cmd := "netductor edge vpn-ensure " + ShellQuote(o.DeviceID)
		out, err := runSSHOnPort(day2SSHPort(), "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, cmd, o.PrimaryKeyPassphrase)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: vpn-ensure on primary:", err, strings.TrimSpace(out))
		} else {
			raw := strings.TrimSpace(out)
			// last JSON object line
			for _, line := range strings.Split(raw, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "{") {
					var links map[string]string
					if json.Unmarshal([]byte(line), &links) == nil {
						vless = strings.TrimSpace(links["vless"])
						if vless == "" {
							vless = strings.TrimSpace(links["subscription"])
						}
					}
				}
				if strings.HasPrefix(line, "vless://") {
					vless = line
				}
			}
		}
	}

	initWait := `#!/bin/sh /etc/rc.common
START=99
USE_PROCD=1
start_service() {
  [ -f /etc/netductor-agent/sing-box-client.json ] || exit 0
  procd_open_instance
  procd_set_param command /usr/sbin/sing-box run -c /etc/netductor-agent/sing-box-client.json
  procd_set_param respawn
  procd_close_instance
}
`
	tmpInit := filepath.Join(os.TempDir(), "nd-netductor-vpn-init")
	_ = os.WriteFile(tmpInit, []byte(initWait), 0o755)
	defer os.Remove(tmpInit)

	if vless == "" {
		fmt.Fprintln(os.Stderr, "==> vpn_client: binary installed; VLESS deferred until agent online + template")
		_ = runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpInit, "/etc/init.d/netductor-vpn", o.PrimaryKeyPassphrase)
		_, _ = runSSHOnPort(port, pass, o.PrimaryKey, user, host, "chmod 755 /etc/init.d/netductor-vpn; /etc/init.d/netductor-vpn enable", o.PrimaryKeyPassphrase)
		return nil
	}

	fmt.Fprintln(os.Stderr, "==> vpn_client: write client config on router")
	cfgJSON, err := edgeagent.VLESSClientConfigOpts(vless, edgeagent.ClientOpts{
		Mode: "tun", SoftFallback: true, PrimaryHost: o.PrimaryHost, DNSMode: "vpn",
	})
	if err != nil {
		return fmt.Errorf("vless config: %w", err)
	}
	tmpV := filepath.Join(os.TempDir(), "nd-vpn.vless")
	tmpC := filepath.Join(os.TempDir(), "nd-sing-box-client.json")
	_ = os.WriteFile(tmpV, []byte(vless+"\n"), 0o600)
	_ = os.WriteFile(tmpC, cfgJSON, 0o600)
	defer os.Remove(tmpV)
	defer os.Remove(tmpC)
	_, _ = runSSHOnPort(port, pass, o.PrimaryKey, user, host, "mkdir -p /etc/netductor-agent", o.PrimaryKeyPassphrase)
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpV, "/etc/netductor-agent/vpn.vless", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp vless: %w", err)
	}
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpC, "/etc/netductor-agent/sing-box-client.json", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp config: %w", err)
	}
	_ = runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpInit, "/etc/init.d/netductor-vpn", o.PrimaryKeyPassphrase)
	_, _ = runSSHOnPort(port, pass, o.PrimaryKey, user, host, "chmod 755 /etc/init.d/netductor-vpn; /etc/init.d/netductor-vpn enable", o.PrimaryKeyPassphrase)
	fmt.Fprintln(os.Stderr, "==> vpn_client: sing-box + VLESS staged (service starts when config present after reboot)")
	return nil
}
