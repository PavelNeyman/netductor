package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/PavelNeyman/netductor/internal/download"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edgeagent"
)

func configBackup(client *http.Client, cfg config) string {
	tmp := filepath.Join(os.TempDir(), "nd-cfg-"+cfg.DeviceID+".tar.gz")
	_ = os.Remove(tmp)
	cmd := exec.Command("tar", "-czf", tmp, "-C", "/etc", "config")
	if out, err := cmd.CombinedOutput(); err != nil {
		// fallback busybox
		cmd = exec.Command("tar", "-czf", tmp, "/etc/config")
		out2, err2 := cmd.CombinedOutput()
		if err2 != nil {
			return "tar failed: " + truncate(string(out)+" "+string(out2), 500)
		}
	}
	defer os.Remove(tmp)
	f, err := os.Open(tmp)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	req, err := http.NewRequest(http.MethodPost, cfg.Server+"/api/edge/backup", f)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/gzip")
	req.Header.Set("X-Device-ID", cfg.DeviceID)
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Sprintf("upload HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return "uploaded: " + truncate(string(body), 300)
}

func agentUpdate(arg string) string {
	url, wantSHA, _ := edgeagent.SplitArg(arg)
	if url == "" {
		return "agent_update: need URL|sha256|confirm=yes"
	}
	if err := requireSHA256Hex(wantSHA); err != nil {
		return "agent_update: " + err.Error()
	}
	dest := "/usr/sbin/netductor-agent"
	if _, err := os.Stat(dest); err != nil {
		dest = "/usr/bin/netductor-agent"
	}
	// Stage on same directory as dest so rename is atomic when possible.
	tmp, err := os.CreateTemp(filepath.Dir(dest), "netductor-agent.*.new")
	if err != nil {
		// fall back to /tmp then copy
		tmp, err = os.CreateTemp(os.TempDir(), "netductor-agent.*.new")
		if err != nil {
			return "agent_update: temp: " + err.Error()
		}
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath) // downloadFile creates the file
	if err := downloadFile(url, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "download: " + err.Error()
	}
	sum, err := fileSHA256(tmpPath)
	if err != nil || !strings.EqualFold(sum, wantSHA) {
		_ = os.Remove(tmpPath)
		return fmt.Sprintf("sha256 mismatch got=%s want=%s", sum, wantSHA)
	}
	if !isELF(tmpPath) {
		_ = os.Remove(tmpPath)
		return "agent_update: not an ELF binary"
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return "agent_update: chmod: " + err.Error()
	}
	prev := dest + ".prev"
	hadDest := false
	if _, err := os.Stat(dest); err == nil {
		hadDest = true
		_ = os.Remove(prev) // best-effort old prev only after we know dest exists
		if err := os.Rename(dest, prev); err != nil {
			_ = os.Remove(tmpPath)
			return "agent_update: cannot save prev: " + err.Error()
		}
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		// Cross-filesystem: stage a second temp in dest dir, copy, verify, atomic rename.
		stage, serr := os.CreateTemp(filepath.Dir(dest), "netductor-agent.*.stage")
		if serr != nil {
			_ = os.Remove(tmpPath)
			if hadDest {
				_ = os.Rename(prev, dest)
			}
			return "agent_update: stage: " + serr.Error()
		}
		stagePath := stage.Name()
		in, rerr := os.Open(tmpPath)
		if rerr != nil {
			_ = stage.Close()
			_ = os.Remove(stagePath)
			_ = os.Remove(tmpPath)
			if hadDest {
				_ = os.Rename(prev, dest)
			}
			return "agent_update: " + rerr.Error()
		}
		_, cerr := io.Copy(stage, in)
		_ = in.Close()
		if cerr != nil {
			_ = stage.Close()
			_ = os.Remove(stagePath)
			_ = os.Remove(tmpPath)
			if hadDest {
				_ = os.Rename(prev, dest)
			}
			return "agent_update: copy: " + cerr.Error()
		}
		_ = stage.Sync()
		_ = stage.Close()
		_ = os.Remove(tmpPath)
		sum2, herr := fileSHA256(stagePath)
		if herr != nil || !strings.EqualFold(sum2, wantSHA) {
			_ = os.Remove(stagePath)
			if hadDest {
				_ = os.Rename(prev, dest)
			}
			return "agent_update: stage sha mismatch"
		}
		if !isELF(stagePath) {
			_ = os.Remove(stagePath)
			if hadDest {
				_ = os.Rename(prev, dest)
			}
			return "agent_update: stage not ELF"
		}
		_ = os.Chmod(stagePath, 0o755)
		if err2 := os.Rename(stagePath, dest); err2 != nil {
			_ = os.Remove(stagePath)
			if hadDest {
				if r2 := os.Rename(prev, dest); r2 != nil {
					return "agent_update: final rename failed and rollback failed: " + err2.Error() + "; " + r2.Error()
				}
			}
			return "agent_update: final rename: " + err2.Error()
		}
	}
	return "updated " + dest + " (prev=" + prev + ") — restart agent to run new binary"
}

func doSysupgrade(arg string) string {
	// URL|sha256|confirm=yes
	url, wantSHA, rest := edgeagent.SplitArg(arg)
	if url == "" {
		return "sysupgrade: URL|sha256|confirm=yes"
	}
	if !edgeagent.SysupgradeAllowed(arg) {
		return "sysupgrade refused: add confirm=yes"
	}
	if err := requireSHA256Hex(wantSHA); err != nil {
		return "sysupgrade: " + err.Error()
	}
	_ = rest
	if _, err := exec.LookPath("sysupgrade"); err != nil {
		return "sysupgrade binary not found"
	}
	img := filepath.Join(os.TempDir(), "nd-firmware.bin")
	if err := downloadFile(url, img); err != nil {
		return "download: " + err.Error()
	}
	sum, err := fileSHA256(img)
	if err != nil || !strings.EqualFold(sum, wantSHA) {
		_ = os.Remove(img)
		return fmt.Sprintf("sha256 mismatch got=%s want=%s", sum, wantSHA)
	}
	// OpenWrt: default sysupgrade KEEPS config. Flag -n means DO NOT keep (factory-like).
	// Never use -n here — that would wipe LAN/Wi-Fi/Netductor settings.
	go func() {
		time.Sleep(3 * time.Second)
		_ = exec.Command("sysupgrade", img).Run()
	}()
	return "sysupgrade scheduled (config preserved; image " + img + ")"
}

func downloadFile(rawURL, dest string) error {
	return download.Get(rawURL, dest, download.Options{
		MaxBytes:         256 << 20, // firmware can be large but hard-capped
		AllowHTTPPrivate: true,      // F20: http only to private/loopback IP
		UserAgent:        "netductor-agent",
	})
}

func requireSHA256Hex(s string) error {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) != 64 {
		return fmt.Errorf("sha256 required (64 hex chars)")
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return fmt.Errorf("sha256 must be hex")
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func applyTemplate(client *http.Client, cfg config) string {
	url := fmt.Sprintf("%s/api/edge/template?device_id=%s", cfg.Server, cfg.DeviceID)
	data, err := doJSON(client, http.MethodGet, url, cfg.Token, nil)
	if err != nil {
		return "template: " + err.Error()
	}
	var tmpl map[string]any
	if json.Unmarshal(data, &tmpl) != nil {
		return "bad template json"
	}
	desired := edgeagent.DesiredUCI(tmpl)
	current := map[string]string{}
	for _, line := range desired {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out, err := exec.Command("uci", "-q", "get", parts[0]).CombinedOutput()
		if err == nil {
			current[parts[0]] = strings.TrimSpace(string(out))
		}
	}
	toSet := edgeagent.DiffUCI(desired, current)
	toSet = skipMissingRadios(toSet)
	if len(toSet) == 0 {
		vpnNote := applyVPNClient(tmpl, cfg)
		gNote := applyGuestFromTemplate(tmpl)
		if strings.HasPrefix(gNote, "template:") {
			return gNote
		}
		return edgeagent.FormatApplyReport(toSet) + vpnNote + gNote
	}
	var errs []string
	for _, line := range toSet {
		if out, err := exec.Command("uci", "set", line).CombinedOutput(); err != nil {
			errs = append(errs, line+": "+strings.TrimSpace(string(out))+" "+err.Error())
		}
	}
	if out, err := exec.Command("uci", "commit").CombinedOutput(); err != nil {
		errs = append(errs, "commit: "+strings.TrimSpace(string(out))+" "+err.Error())
	}
	if len(errs) > 0 {
		return "template: " + strings.Join(errs, "; ")
	}
	if out, err := exec.Command("/etc/init.d/network", "reload").CombinedOutput(); err != nil {
		return "template: network reload: " + strings.TrimSpace(string(out)) + " " + err.Error()
	}
	if out, err := exec.Command("wifi", "reload").CombinedOutput(); err != nil {
		return "template: wifi reload: " + strings.TrimSpace(string(out)) + " " + err.Error()
	}
	vpnNote := applyVPNClient(tmpl, cfg)
	gNote := applyGuestFromTemplate(tmpl)
	if strings.HasPrefix(gNote, "template:") {
		return gNote
	}
	return edgeagent.FormatApplyReport(toSet) + "\n" + strings.Join(toSet, "\n") + vpnNote + gNote
}

func stopEdgeVPN() string {
	note := ""
	if _, err := os.Stat("/etc/init.d/netductor-vpn"); err == nil {
		_ = exec.Command("/etc/init.d/netductor-vpn", "stop").Run()
		_ = exec.Command("/etc/init.d/netductor-vpn", "disable").Run()
		note = "; netductor-vpn stopped"
	}
	return note
}

func vpnFlagTruthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		return s == "1" || s == "true" || s == "yes" || s == "on"
	case float64:
		return x != 0
	default:
		return false
	}
}

func applyVPNClient(tmpl map[string]any, cfg config) string {
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn == nil {
		return stopEdgeVPN()
	}
	enabled := vpnFlagTruthy(vpn["enabled"])
	mode, _ := vpn["mode"].(string)
	mode = strings.ToLower(strings.TrimSpace(mode))
	// enabled=false or mode=off → tear down client (do not leave stale TUN/proxy).
	if !enabled || mode == "off" {
		why := "disabled"
		if enabled && mode == "off" {
			why = "mode=off"
		}
		return "\nvpn " + why + stopEdgeVPN()
	}
	_ = os.MkdirAll("/etc/netductor-agent", 0o700)
	if sub, _ := vpn["subscription"].(string); sub != "" {
		_ = os.WriteFile("/etc/netductor-agent/vpn.subscription", []byte(sub+"\n"), 0o600)
	}
	vless, _ := vpn["vless"].(string)
	if vless != "" {
		_ = os.WriteFile("/etc/netductor-agent/vpn.vless", []byte(vless+"\n"), 0o600)
	}
	note := "\nvpn links saved"
	if vless == "" {
		return note + "; no vless link in template" + stopEdgeVPN()
	}
	if mode == "" {
		mode = "tun"
	}
	dnsMode, _ := vpn["dns"].(string)
	if dnsMode == "" {
		dnsMode, _ = vpn["dns_mode"].(string)
	}
	if dnsMode == "" {
		dnsMode = "vpn"
	}
	// soft_fallback: explicit wins; else fallback=block/none/off ⇒ no ISP urltest.
	soft := true
	if fb, _ := vpn["fallback"].(string); strings.EqualFold(fb, "block") || fb == "none" || fb == "off" {
		soft = false
	}
	if v, ok := vpn["soft_fallback"]; ok {
		soft = vpnFlagTruthy(v)
	}
	primaryHost := ""
	if ph, _ := vpn["primary_host"].(string); ph != "" {
		primaryHost = ph
	}
	if primaryHost == "" && cfg.Server != "" {
		if u, err := url.Parse(cfg.Server); err == nil {
			primaryHost = u.Hostname()
		}
	}
	boxJSON, err := edgeagent.VLESSClientConfigOpts(vless, edgeagent.ClientOpts{
		Mode: mode, SoftFallback: soft, PrimaryHost: primaryHost, DNSMode: dnsMode,
	})
	if err != nil {
		return note + "; vless parse: " + err.Error()
	}
	path := "/etc/netductor-agent/sing-box-client.json"
	_ = os.WriteFile(path, boxJSON, 0o600)
	note += "; config " + path + " mode=" + mode
	bin, err := edgeagent.EnsureSingBox("/usr/sbin/sing-box")
	if err != nil {
		return note + "; sing-box download: " + err.Error()
	}
	note += "; bin " + bin
	init := "#!/bin/sh /etc/rc.common\nSTART=99\nUSE_PROCD=1\nstart_service() {\n  procd_open_instance\n  procd_set_param command " + bin + " run -c /etc/netductor-agent/sing-box-client.json\n  procd_set_param respawn\n  procd_close_instance\n}\n"
	if _, err := os.Stat("/etc/rc.common"); err == nil {
		_ = os.WriteFile("/etc/init.d/netductor-vpn", []byte(init), 0o755)
		_ = exec.Command("/etc/init.d/netductor-vpn", "enable").Run()
		_ = exec.Command("/etc/init.d/netductor-vpn", "restart").Run()
		note += "; netductor-vpn restarted"
		if mode == "tun" {
			time.Sleep(2 * time.Second)
			out, _ := exec.Command("/etc/init.d/netductor-vpn", "status").CombinedOutput()
			st := string(out)
			if strings.Contains(st, "inactive") || strings.Contains(st, "failed") || strings.Contains(strings.ToLower(st), "not running") {
				note += "; tun failed → socks fallback"
				mode = "socks"
				if cfg2, err2 := edgeagent.VLESSClientConfigOpts(vless, edgeagent.ClientOpts{Mode: mode, SoftFallback: soft, PrimaryHost: primaryHost, DNSMode: dnsMode}); err2 == nil {
					_ = os.WriteFile(path, cfg2, 0o600)
					_ = exec.Command("/etc/init.d/netductor-vpn", "restart").Run()
				}
			}
		}
	} else {
		_ = exec.Command(bin, "run", "-c", path).Start()
		note += "; sing-box started"
	}
	if mode == "socks" || mode == "mixed" {
		note += "; local proxy 127.0.0.1:7890 (or NETDUCTOR_EDGE_MIXED_LISTEN)"
	}
	return note
}

func configRestore(client *http.Client, cfg config, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "config_restore: need backup filename"
	}
	safeName, err := validBackupName(name)
	if err != nil {
		return "config_restore: invalid name"
	}
	url := fmt.Sprintf("%s/api/edge/backups?device_id=%s&name=%s", cfg.Server, cfg.DeviceID, safeName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	body, err := readBodyLimited(resp.Body, 64<<20)
	if err != nil {
		return "config_restore: " + err.Error()
	}
	tmp := filepath.Join(os.TempDir(), "nd-restore.tar.gz")
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err.Error()
	}
	defer os.Remove(tmp)
	// configBackup uses: tar -C /etc config → members config/network, …
	// Extract into /etc with strip of the config/ prefix component.
	opts := tarScanOpts{RequirePrefix: "config/"}
	if err := extractTarSafeOpts(tmp, "/etc", opts, []string{"--strip-components=1"}); err != nil {
		return "config_restore: " + err.Error()
	}
	if out, err := exec.Command("uci", "commit").CombinedOutput(); err != nil {
		return "config_restore: uci commit: " + truncate(string(out)+" "+err.Error(), 200)
	}
	return "restored " + safeName + " + uci commit"
}

func storageBackupB(client *http.Client, cfg config) string {
	tmp := filepath.Join(os.TempDir(), "nd-overlay-"+cfg.DeviceID+".tar.gz")
	_ = os.Remove(tmp)
	src := "/overlay"
	if _, err := os.Stat(src); err != nil {
		src = "/etc"
	}
	if out, err := exec.Command("tar", "-czf", tmp, "-C", src, ".").CombinedOutput(); err != nil {
		return "tar: " + truncate(string(out)+" "+err.Error(), 400)
	}
	defer os.Remove(tmp)
	f, err := os.Open(tmp)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	url := strings.TrimRight(cfg.Server, "/") + "/api/edge/storage/backup-push?kind=B&device_id=" + cfg.DeviceID
	req, err := http.NewRequest(http.MethodPost, url, f)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/gzip")
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Sprintf("upload HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return "storage B uploaded: " + truncate(string(body), 300)
}

func storageRestoreLocal(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "storage_restore: need path to nd-restore dir or tar.gz"
	}
	if strings.Contains(arg, "..") || strings.ContainsAny(arg, "\r\n") {
		return "storage_restore: invalid path"
	}
	if !strings.HasSuffix(arg, ".tar.gz") && !strings.HasSuffix(arg, ".tgz") {
		cand := filepath.Join(arg, "overlay-B.tar.gz")
		if _, err := os.Stat(cand); err == nil {
			arg = cand
		}
	}
	// Only archives under USB mount or /tmp (operator-staged).
	okRoot := strings.HasPrefix(arg, "/mnt/netductor/") || strings.HasPrefix(arg, "/tmp/")
	if !okRoot {
		return "storage_restore: path must be under /mnt/netductor or /tmp"
	}
	if _, err := os.Stat("/mnt/netductor"); err != nil {
		return "storage_restore: mount USB at /mnt/netductor first"
	}
	if _, err := os.Stat(arg); err != nil {
		return "storage_restore: archive not found"
	}
	if err := extractTarSafe(arg, "/mnt/netductor"); err != nil {
		return "storage_restore: " + err.Error()
	}
	pin := "/etc/netductor-agent/pin"
	if _, err := os.Stat(pin); err == nil {
		_ = exec.Command("cp", "-a", pin+"/.", "/mnt/netductor/").Run()
	}
	return "restore extracted; merge pin attempted — reboot when ready"
}

func storageRestoreRemote(client *http.Client, cfg config, arg string) string {
	// arg: filename optional; empty = latest via /api/edge/storage/latest-b
	name := strings.TrimSpace(arg)
	var url string
	if name == "" {
		url = strings.TrimRight(cfg.Server, "/") + "/api/edge/storage/latest-b?device_id=" + cfg.DeviceID
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err.Error()
		}
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
		resp, err := client.Do(req)
		if err != nil {
			return err.Error()
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Sprintf("latest-b HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		}
		var meta map[string]any
		_ = json.Unmarshal(body, &meta)
		name, _ = meta["file"].(string)
		if name == "" {
			return "latest-b: no file"
		}
	}
	safeName, err := validBackupName(name)
	if err != nil {
		return "storage_restore_remote: invalid name"
	}
	name = safeName
	// Prefer edgestorage path; also try legacy edge backups list
	dl := strings.TrimRight(cfg.Server, "/") + "/api/edge/storage/download?device_id=" + cfg.DeviceID + "&name=" + name
	req, err := http.NewRequest(http.MethodGet, dl, nil)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("download HTTP %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	body, err := readBodyLimited(resp.Body, 512<<20)
	if err != nil {
		return "storage_restore_remote: " + err.Error()
	}
	tmp := filepath.Join(os.TempDir(), "nd-B-restore.tar.gz")
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err.Error()
	}
	defer os.Remove(tmp)
	_ = os.MkdirAll("/mnt/netductor", 0o755)
	// ensure mount
	_ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor || (block info 2>/dev/null; for p in /dev/sda1 /dev/sda /dev/sdb1; do [ -b $p ] && mount $p /mnt/netductor && break; done)").Run()
	if err := extractTarSafe(tmp, "/mnt/netductor"); err != nil {
		return "storage_restore_remote: " + err.Error()
	}
	pin := "/etc/netductor-agent/pin"
	if _, err := os.Stat(pin); err == nil {
		_ = exec.Command("cp", "-a", pin+"/.", "/mnt/netductor/").Run()
	}
	_ = os.WriteFile("/etc/netductor-agent/restore_ready", []byte("1\n"), 0o644)
	return "remote B restored to /mnt/netductor; pin merged — reboot when ready"
}

func storageInitUSB(arg string) string {
	// arg may contain format=yes
	format := strings.Contains(arg, "format=yes")
	script := `
set -e
DEV=
for d in /sys/block/sd*; do
  [ -d "$d" ] || continue
  rem=$(cat "$d/removable" 2>/dev/null || echo 0)
  [ "$rem" = "1" ] || continue
  DEV=$(basename "$d")
  break
done
[ -n "$DEV" ] || { echo no_removable_usb; exit 1; }
PART=/dev/${DEV}1
DISK=/dev/$DEV
`
	if format {
		script += `
if [ ! -b "$PART" ]; then
  command -v parted >/dev/null 2>&1 && parted -s $DISK -- mklabel gpt mkpart primary ext4 1MiB 100% || true
  sleep 2
fi
[ -b "$PART" ] || PART=$DISK
umount "$PART" 2>/dev/null || true
mkfs.ext4 -F -L netductor "$PART"
`
	}
	script += `
[ -b "$PART" ] || PART=$DISK
mkdir -p /mnt/netductor
umount /mnt/netductor 2>/dev/null || true
mount "$PART" /mnt/netductor
UUID=$(block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
[ -n "$UUID" ] || UUID=$(blkid -s UUID -o value "$PART" 2>/dev/null || true)
if [ -n "$UUID" ]; then
  uci -q delete fstab.nd_extroot
  uci set fstab.nd_extroot=mount
  uci set fstab.nd_extroot.uuid="$UUID"
  uci set fstab.nd_extroot.target=/overlay
  uci set fstab.nd_extroot.enabled=1
  uci commit fstab
fi
echo 1 > /etc/netductor-agent/expect_usb
echo INIT_OK part=$PART uuid=$UUID
`
	out, err := exec.Command("sh", "-c", script).CombinedOutput()
	if err != nil {
		return truncate(string(out)+" "+err.Error(), 500)
	}
	return truncate(string(out), 500)
}

func skipMissingRadios(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		key := line
		if i := strings.Index(line, "="); i > 0 {
			key = line[:i]
		}
		radio := ""
		switch {
		case strings.HasPrefix(key, "wireless.radio0") || strings.HasPrefix(key, "wireless.default_radio0"):
			radio = "wireless.radio0"
		case strings.HasPrefix(key, "wireless.radio1") || strings.HasPrefix(key, "wireless.default_radio1"):
			radio = "wireless.radio1"
		}
		if radio != "" {
			if err := exec.Command("uci", "-q", "get", radio).Run(); err != nil {
				continue
			}
		}
		out = append(out, line)
	}
	return out
}
