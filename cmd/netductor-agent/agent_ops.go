package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		return "agent_update: need URL or URL|sha256"
	}
	tmp := filepath.Join(os.TempDir(), "netductor-agent.new")
	if err := downloadFile(url, tmp); err != nil {
		return "download: " + err.Error()
	}
	if wantSHA != "" {
		sum, err := fileSHA256(tmp)
		if err != nil || !strings.EqualFold(sum, wantSHA) {
			_ = os.Remove(tmp)
			return fmt.Sprintf("sha256 mismatch got=%s want=%s", sum, wantSHA)
		}
	}
	_ = os.Chmod(tmp, 0o755)
	dest := "/usr/sbin/netductor-agent"
	if _, err := os.Stat(dest); err != nil {
		dest = "/usr/bin/netductor-agent"
	}
	if err := os.Rename(tmp, dest); err != nil {
		// cross-device
		in, _ := os.ReadFile(tmp)
		if err2 := os.WriteFile(dest, in, 0o755); err2 != nil {
			return err2.Error()
		}
		_ = os.Remove(tmp)
	}
	return "updated " + dest + " — restart agent to run new binary"
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
	_ = rest
	if _, err := exec.LookPath("sysupgrade"); err != nil {
		return "sysupgrade binary not found"
	}
	img := filepath.Join(os.TempDir(), "nd-firmware.bin")
	if err := downloadFile(url, img); err != nil {
		return "download: " + err.Error()
	}
	if wantSHA != "" {
		sum, err := fileSHA256(img)
		if err != nil || !strings.EqualFold(sum, wantSHA) {
			_ = os.Remove(img)
			return fmt.Sprintf("sha256 mismatch got=%s want=%s", sum, wantSHA)
		}
	}
	// -n keep config by default
	go func() {
		time.Sleep(3 * time.Second)
		_ = exec.Command("sysupgrade", "-n", img).Run()
	}()
	return "sysupgrade scheduled (keep config flags: default -n keep; image " + img + ")"
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
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
	if len(toSet) == 0 {
		return edgeagent.FormatApplyReport(toSet) + applyVPNClient(tmpl) + applyGuestFromTemplate(tmpl)
	}
	for _, line := range toSet {
		_ = exec.Command("uci", "set", line).Run()
	}
	_ = exec.Command("uci", "commit").Run()
	_ = exec.Command("/etc/init.d/network", "reload").Run()
	_ = exec.Command("wifi", "reload").Run()
	vpnNote := applyVPNClient(tmpl)
	gNote := applyGuestFromTemplate(tmpl)
	return edgeagent.FormatApplyReport(toSet) + "\n" + strings.Join(toSet, "\n") + vpnNote + gNote
}

func applyVPNClient(tmpl map[string]any) string {
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn == nil {
		return ""
	}
	enabled := false
	switch v := vpn["enabled"].(type) {
	case bool:
		enabled = v
	case string:
		enabled = v == "true" || v == "1"
	}
	if !enabled {
		return ""
	}
	_ = os.MkdirAll("/etc/netductor-agent", 0o700)
	if sub, _ := vpn["subscription"].(string); sub != "" {
		_ = os.WriteFile("/etc/netductor-agent/vpn.subscription", []byte(sub+"\n"), 0o600)
	}
	vless, _ := vpn["vless"].(string)
	hy2, _ := vpn["hy2"].(string)
	if vless != "" {
		_ = os.WriteFile("/etc/netductor-agent/vpn.vless", []byte(vless+"\n"), 0o600)
	}
	if hy2 != "" {
		_ = os.WriteFile("/etc/netductor-agent/vpn.hy2", []byte(hy2+"\n"), 0o600)
	}
	note := "\nvpn links saved"
	if vless == "" {
		return note
	}
	mode, _ := vpn["mode"].(string)
	if mode == "" {
		mode = "socks" // default: no TUN required
	}
	cfg, err := edgeagent.VLESSClientConfig(vless, mode)
	if err != nil {
		return note + "; vless parse: " + err.Error()
	}
	path := "/etc/netductor-agent/sing-box-client.json"
	_ = os.WriteFile(path, cfg, 0o600)
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
	} else {
		_ = exec.Command(bin, "run", "-c", path).Start()
		note += "; sing-box started"
	}
	if mode == "socks" {
		note += "; local proxy 0.0.0.0:7890 (set LAN devices or transparent redirect manually)"
	}
	return note
}

func configRestore(client *http.Client, cfg config, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "config_restore: need backup filename"
	}
	url := fmt.Sprintf("%s/api/edge/backups?device_id=%s&name=%s", cfg.Server, cfg.DeviceID, name)
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
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	tmp := filepath.Join(os.TempDir(), "nd-restore.tar.gz")
	f, err := os.Create(tmp)
	if err != nil {
		return err.Error()
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return err.Error()
	}
	defer os.Remove(tmp)
	// extract into /etc/config
	_ = os.MkdirAll("/etc/config", 0o755)
	out, err := exec.Command("tar", "-xzf", tmp, "-C", "/etc").CombinedOutput()
	if err != nil {
		out2, err2 := exec.Command("tar", "-xzf", tmp, "-C", "/").CombinedOutput()
		if err2 != nil {
			return "extract: " + truncate(string(out)+" "+string(out2), 500)
		}
	}
	_ = exec.Command("uci", "commit").Run()
	return "restored " + name + " + uci commit"
}

