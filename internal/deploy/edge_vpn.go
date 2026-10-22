package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edgeagent"
)

// OpenWrt storage practice (travel / 16MB class):
//
//  1. extroot — move /overlay to USB (module overlay_ext). Best when many packages need space.
//     Official OpenWrt path; if USB missing at boot, device falls back to internal overlay.
//  2. Data mount — keep system on internal flash; put large binaries on USB (e.g. /mnt/.../netductor/).
//     Safer if USB is optional day-to-day; only VPN payload depends on the stick.
//  3. Do not default to "split fancy partitions" on first deploy — one ext4 volume is enough:
//     either full extroot or a single data mount for netductor bins.
//
// vpn_client uses (2) when internal free space < sing-box size; never silent-success on ENOSPC.

const singBoxMarginKB int64 = 512 // leave headroom for keys/config after binary

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
	st, err := os.Stat(localBin)
	if err != nil {
		return fmt.Errorf("sing-box local: %w", err)
	}
	needKB := st.Size()/1024 + singBoxMarginKB
	if needKB < 1024 {
		needKB = 1024
	}

	pass := o.RouterPass
	user := o.RouterUser
	host := o.RouterHost
	port := factorySSHPort()

	dest, err := pickSingBoxInstallPath(port, pass, o.PrimaryKey, user, host, o.PrimaryKeyPassphrase, needKB)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "==> vpn_client: install path %s (need ≥%d KiB free)\n", dest, needKB)

	remoteDir := filepath.Dir(dest)
	if remoteDir != "/" && remoteDir != "." {
		if _, err := runSSHOnPort(port, pass, o.PrimaryKey, user, host, "mkdir -p "+ShellQuote(remoteDir), o.PrimaryKeyPassphrase); err != nil {
			return fmt.Errorf("mkdir %s: %w", remoteDir, err)
		}
	}
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, localBin, dest, o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp sing-box → %s: %w", dest, err)
	}
	chmodCmd := "chmod 755 " + ShellQuote(dest)
	// Symlink into /usr/sbin when binary lives on USB/data mount so procd path stays stable.
	if dest != "/usr/sbin/sing-box" {
		chmodCmd += "; ln -sfn " + ShellQuote(dest) + " /usr/sbin/sing-box"
	}
	if _, err := runSSHOnPort(port, pass, o.PrimaryKey, user, host, chmodCmd, o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("chmod/symlink sing-box: %w", err)
	}

	vless := ""
	if o.PrimaryHost != "" && o.PrimaryKey != "" {
		fmt.Fprintln(os.Stderr, "==> vpn_client: ensure edge user on primary")
		cmd := "netductor edge vpn-ensure " + ShellQuote(o.DeviceID)
		out, err := runSSHOnPort(day2SSHPort(), "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, cmd, o.PrimaryKeyPassphrase)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: edge vpn-ensure:", err, out)
		} else {
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "{") {
					var links map[string]string
					if json.Unmarshal([]byte(line), &links) == nil {
						if v := strings.TrimSpace(links["vless"]); v != "" {
							vless = v
						}
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
  BIN=/usr/sbin/sing-box
  [ -x "$BIN" ] || BIN=/mnt/netductor/sing-box
  [ -x "$BIN" ] || exit 0
  procd_open_instance
  procd_set_param command "$BIN" run -c /etc/netductor-agent/sing-box-client.json
  procd_set_param respawn
  procd_close_instance
}
`
	vpnTmp, errTmp := os.MkdirTemp("", "nd-edge-vpn-*")
	if errTmp != nil {
		return fmt.Errorf("vpn temp dir: %w", errTmp)
	}
	defer os.RemoveAll(vpnTmp)
	_ = os.Chmod(vpnTmp, 0o700)
	tmpInit := filepath.Join(vpnTmp, "netductor-vpn-init")
	if err := os.WriteFile(tmpInit, []byte(initWait), 0o755); err != nil {
		return fmt.Errorf("write vpn init: %w", err)
	}

	if vless == "" {
		fmt.Fprintln(os.Stderr, "==> vpn_client: binary installed; VLESS deferred until agent online + template")
		if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpInit, "/etc/init.d/netductor-vpn", o.PrimaryKeyPassphrase); err != nil {
			return fmt.Errorf("vpn_client: install init (deferred VLESS): %w", err)
		}
		if _, err := runSSHOnPort(port, pass, o.PrimaryKey, user, host, "chmod 755 /etc/init.d/netductor-vpn; /etc/init.d/netductor-vpn enable", o.PrimaryKeyPassphrase); err != nil {
			return fmt.Errorf("vpn_client: enable init (deferred VLESS): %w", err)
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, "==> vpn_client: write client config on router")
	cfgJSON, err := edgeagent.VLESSClientConfigOpts(vless, edgeagent.ClientOpts{
		Mode: "tun", SoftFallback: true, PrimaryHost: o.PrimaryHost, DNSMode: "vpn",
	})
	if err != nil {
		return fmt.Errorf("vless config: %w", err)
	}
	tmpV := filepath.Join(vpnTmp, "vpn.vless")
	tmpC := filepath.Join(vpnTmp, "sing-box-client.json")
	if err := os.WriteFile(tmpV, []byte(vless+"\n"), 0o600); err != nil {
		return fmt.Errorf("write vless: %w", err)
	}
	if err := os.WriteFile(tmpC, cfgJSON, 0o600); err != nil {
		return fmt.Errorf("write client json: %w", err)
	}
	_, _ = runSSHOnPort(port, pass, o.PrimaryKey, user, host, "mkdir -p /etc/netductor-agent", o.PrimaryKeyPassphrase)
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpV, "/etc/netductor-agent/vpn.vless", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp vless: %w", err)
	}
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpC, "/etc/netductor-agent/sing-box-client.json", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp config: %w", err)
	}
	if err := runSCPOnPort(port, pass, o.PrimaryKey, user, host, tmpInit, "/etc/init.d/netductor-vpn", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("install netductor-vpn init: %w", err)
	}
	if _, err := runSSHOnPort(port, pass, o.PrimaryKey, user, host, "chmod 755 /etc/init.d/netductor-vpn; /etc/init.d/netductor-vpn enable", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("enable netductor-vpn: %w", err)
	}
	fmt.Fprintln(os.Stderr, "==> vpn_client: sing-box + VLESS staged (service starts when config present after reboot)")
	return nil
}

// pickSingBoxInstallPath chooses internal /usr/sbin or a mounted USB data path.
func pickSingBoxInstallPath(port string, pass, key, user, host, keyPass string, needKB int64) (string, error) {
	// Report free space (overlay preferred, then root).
	script := `set +e
ov=$(df -k /overlay 2>/dev/null | tail -1 | awk '{print $4}')
rt=$(df -k / 2>/dev/null | tail -1 | awk '{print $4}')
echo OVERLAY_FREE_KB=${ov:-0}
echo ROOT_FREE_KB=${rt:-0}
# mounted USB-like candidates — only if filesystem device differs from overlay/root
ov_dev=$(df -P /overlay 2>/dev/null | tail -1 | awk '{print $1}')
rt_dev=$(df -P / 2>/dev/null | tail -1 | awk '{print $1}')
for m in /mnt/netductor /mnt/usb /mnt/sda1 /mnt/sda /mnt/mmcblk0p1 /opt; do
  [ -d "$m" ] || continue
  line=$(df -Pk "$m" 2>/dev/null | tail -1)
  [ -n "$line" ] || continue
  m_dev=$(echo "$line" | awk '{print $1}')
  fre=$(echo "$line" | awk '{print $4}')
  [ -n "$fre" ] || continue
  # skip if same block device as overlay or root (directory on internal flash)
  if [ -n "$ov_dev" ] && [ "$m_dev" = "$ov_dev" ]; then continue; fi
  if [ -n "$rt_dev" ] && [ "$m_dev" = "$rt_dev" ]; then continue; fi
  echo MOUNT:$m:$fre:$m_dev
done
# unmounted block devices hint (not auto-mounted)
if command -v block >/dev/null 2>&1; then
  block info 2>/dev/null | head -20 | sed 's/^/BLOCKINFO:/'
fi
true
`
	out, err := runSSHOnPort(port, pass, key, user, host, script, keyPass)
	if err != nil {
		return "", fmt.Errorf("space probe: %w (%s)", err, strings.TrimSpace(out))
	}
	var overlayFree, rootFree int64
	var mounts []struct {
		path string
		free int64
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "OVERLAY_FREE_KB="):
			overlayFree, _ = strconv.ParseInt(strings.TrimPrefix(line, "OVERLAY_FREE_KB="), 10, 64)
		case strings.HasPrefix(line, "ROOT_FREE_KB="):
			rootFree, _ = strconv.ParseInt(strings.TrimPrefix(line, "ROOT_FREE_KB="), 10, 64)
		case strings.HasPrefix(line, "MOUNT:"):
			parts := strings.Split(strings.TrimPrefix(line, "MOUNT:"), ":")
			// MOUNT:path:freeKB or MOUNT:path:freeKB:device
			if len(parts) >= 2 {
				freIdx := 1
				if len(parts) >= 3 {
					// path may contain colons rarely; free is first purely numeric field after path
					for i := 1; i < len(parts); i++ {
						if _, e := strconv.ParseInt(parts[i], 10, 64); e == nil {
							freIdx = i
							break
						}
					}
				}
				fre, _ := strconv.ParseInt(parts[freIdx], 10, 64)
				path := strings.Join(parts[:freIdx], ":")
				if fre > 0 && path != "" {
					mounts = append(mounts, struct {
						path string
						free int64
					}{path, fre})
				}
			}
		}
	}
	fmt.Fprintf(os.Stderr, "==> vpn_client: free overlay=%dKiB root=%dKiB usb_mounts=%d need=%dKiB\n",
		overlayFree, rootFree, len(mounts), needKB)

	// Prefer USB/data mounts first (16 MiB flash class cannot host stock sing-box).
	for _, m := range mounts {
		if m.free < needKB {
			continue
		}
		if m.path == "/opt" {
			return "/opt/netductor/sing-box", nil
		}
		if m.path == "/mnt/netductor" {
			return "/mnt/netductor/sing-box", nil
		}
		return filepath.Join(m.path, "netductor", "sing-box"), nil
	}

	// Internal: use overlay free only (df / can be larger than writable /usr).
	// Never use max(root, overlay) — root free does not guarantee /usr/sbin write.
	if overlayFree >= needKB {
		return "/usr/sbin/sing-box", nil
	}

	return "", fmt.Errorf("vpn_client: not enough free space for sing-box (need ≥%d KiB; overlay=%d root=%d usb_mounts=%d). "+
		"On 16MiB flash put sing-box on USB/extroot: (1) overlay_ext so /overlay is large; "+
		"(2) mount USB under /mnt/usb or /mnt/sda1 and re-run; "+
		"(3) free overlay space. Deploy aborted before treating this as success",
		needKB, overlayFree, rootFree, len(mounts))
}
