// netductor-agent — outbound OpenWrt edge agent
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edgeagent"
)

var version = "0.7.0-dev"

type config struct {
	Server   string
	Token    string
	DeviceID string
	Interval int
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Printf("netductor-agent %s\n", version)
			return
		case "help", "-h", "--help":
			fmt.Print(`netductor-agent — outbound edge agent for OpenWrt

Config /etc/netductor-agent/config:
  SERVER=http://vps:8787
  TOKEN=...
  DEVICE_ID=site1
  INTERVAL=60

Commands (from VPS):
  ping, status, metrics
  config_backup          — tar /etc/config → VPS
  uci_get|show|set|commit|batch
  wifi_reload, network_reload, reboot
  agent_update           — arg: URL or URL|sha256
  sysupgrade             — arg: URL|sha256|confirm=yes
`)
			return
		}
	}
	cfg := loadConfig()
	if cfg.Server == "" || cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "SERVER and TOKEN required")
		os.Exit(1)
	}
	cfg.Server = strings.TrimRight(cfg.Server, "/")
	client := &http.Client{Timeout: 120 * time.Second}
	for {
		if err := ensureEnrolled(client, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "enroll: %v\n", err)
			time.Sleep(time.Duration(cfg.Interval) * time.Second)
			continue
		}
		if _, err := os.Stat(filepath.Join(agentDir(), "applied_template")); err != nil {
			res := applyTemplate(client, cfg)
			fmt.Fprintf(os.Stderr, "apply_template: %s\n", res)
			if !strings.HasPrefix(res, "template:") {
				_ = os.WriteFile(filepath.Join(agentDir(), "applied_template"), []byte(res+"\n"), 0o600)
			}
		}
		checkRelayAndFallback()
		if err := heartbeat(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "heartbeat: %v\n", err)
		}
		if err := pollCmds(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "poll: %v\n", err)
		}
		time.Sleep(time.Duration(cfg.Interval) * time.Second)
	}
}

func agentDir() string {
	if v := os.Getenv("NETDUCTOR_AGENT_DIR"); v != "" {
		return v
	}
	return "/etc/netductor-agent"
}

func deviceTokenPath() string {
	return filepath.Join(agentDir(), "device_token")
}

func loadDeviceToken() string {
	b, err := os.ReadFile(deviceTokenPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func saveDeviceToken(tok string) {
	_ = os.MkdirAll("/etc/netductor-agent", 0o700)
	_ = os.WriteFile(deviceTokenPath(), []byte(tok+"\n"), 0o600)
}

// ensureEnrolled uses bootstrap TOKEN until device_token is issued.
func ensureEnrolled(client *http.Client, cfg *config) error {
	if tok := loadDeviceToken(); tok != "" {
		cfg.Token = tok
		return nil
	}
	payload := collectMetrics()
	payload["device_id"] = cfg.DeviceID
	// bootstrap auth uses config TOKEN (edge_bootstrap)
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, cfg.Server+"/api/edge/enroll", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	var out struct {
		Status      string `json:"status"`
		DeviceToken string `json:"device_token"`
	}
	_ = json.Unmarshal(data, &out)
	switch out.Status {
	case "approved":
		if out.DeviceToken != "" {
			saveDeviceToken(out.DeviceToken)
			cfg.Token = out.DeviceToken
		}
		return nil
	case "pending":
		return fmt.Errorf("pending approval")
	case "denied", "revoked":
		return fmt.Errorf("status %s", out.Status)
	default:
		return fmt.Errorf("status %s", out.Status)
	}
}

func loadConfig() config {
	c := config{Interval: 60, DeviceID: hostname()}
	for _, path := range []string{os.Getenv("NETDUCTOR_AGENT_CONF"), "/etc/netductor-agent/config"} {
		if path == "" {
			continue
		}
		if b, err := os.ReadFile(path); err == nil {
			parseKV(string(b), &c)
			break
		}
	}
	if v := os.Getenv("NETDUCTOR_SERVER"); v != "" {
		c.Server = v
	}
	if v := os.Getenv("NETDUCTOR_TOKEN"); v != "" {
		c.Token = v
	}
	if v := os.Getenv("NETDUCTOR_DEVICE_ID"); v != "" {
		c.DeviceID = v
	}
	if v := os.Getenv("NETDUCTOR_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.Interval = n
		}
	}
	if c.DeviceID == "" {
		c.DeviceID = hostname()
	}
	return c
}

func parseKV(s string, c *config) {
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "SERVER":
			c.Server = strings.TrimSpace(v)
		case "TOKEN":
			c.Token = strings.TrimSpace(v)
		case "DEVICE_ID":
			c.DeviceID = strings.TrimSpace(v)
		case "INTERVAL":
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				c.Interval = n
			}
		}
	}
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func boardName() string {
	for _, p := range []string{"/tmp/sysinfo/model", "/proc/device-tree/model"} {
		if b, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

func doJSON(client *http.Client, method, url, token string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return data, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	return data, nil
}

func collectMetrics() map[string]any {
	m := map[string]any{
		"hostname": hostname(),
		"board":    boardName(),
		"agent":    version,
		"ts":       time.Now().Unix(),
	}
	// uptime
	if b, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(b))
		if len(fields) > 0 {
			m["uptime_sec"], _ = strconv.ParseFloat(fields[0], 64)
		}
	}
	// mem
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail float64
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %f", &total)
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(line, "MemAvailable: %f", &avail)
			}
		}
		if total > 0 {
			m["mem_total_kb"] = total
			m["mem_avail_kb"] = avail
			m["mem_pct"] = (1 - avail/total) * 100
		}
	}
	// load
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		f := strings.Fields(string(b))
		if len(f) >= 3 {
			m["load"] = map[string]string{"1": f[0], "5": f[1], "15": f[2]}
		}
	}
	// WAN IP
	if out, err := exec.Command("ip", "-4", "route", "get", "1.1.1.1").Output(); err == nil {
		// ... src x.x.x.x
		parts := strings.Fields(string(out))
		for i, p := range parts {
			if p == "src" && i+1 < len(parts) {
				m["wan_ip"] = parts[i+1]
				break
			}
		}
	}
	// wifi clients rough
	if out, err := exec.Command("iwinfo").Output(); err == nil {
		m["iwinfo"] = truncate(string(out), 1500)
	}
	return m
}


func checkRelayAndFallback() {
	relayHost := strings.TrimSpace(readFirstLine("/etc/netductor-agent/relay.host"))
	if relayHost == "" {
		return
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	c, err := d.Dial("tcp", net.JoinHostPort(relayHost, "443"))
	ok := err == nil
	if ok {
		_ = c.Close()
	}
	statePath := "/etc/netductor-agent/relay.health"
	prev, _ := os.ReadFile(statePath)
	prevOK := strings.TrimSpace(string(prev)) == "ok"
	if ok {
		_ = os.WriteFile(statePath, append([]byte("ok"), 10), 0o644)
		if !prevOK {
			_ = exec.Command("/etc/init.d/netductor-vpn", "start").Run()
		}
		return
	}
	_ = os.WriteFile(statePath, append([]byte("fail"), 10), 0o644)
	if prevOK || len(prev) == 0 {
		_ = exec.Command("/etc/init.d/netductor-vpn", "stop").Run()
	}
}

func readFirstLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := string(b)
	if i := strings.IndexByte(s, 10); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func heartbeat(client *http.Client, cfg config) error {
	payload := collectMetrics()
	payload["device_id"] = cfg.DeviceID
	_, err := doJSON(client, http.MethodPost, cfg.Server+"/api/edge/heartbeat", cfg.Token, payload)
	if err != nil {
		return err
	}
	// also store metrics history on VPS
	_, _ = doJSON(client, http.MethodPost, cfg.Server+"/api/edge/metrics", cfg.Token, payload)
	return nil
}

func pollCmds(client *http.Client, cfg config) error {
	url := fmt.Sprintf("%s/api/edge/commands?device_id=%s", cfg.Server, cfg.DeviceID)
	data, err := doJSON(client, http.MethodGet, url, cfg.Token, nil)
	if err != nil {
		return err
	}
	var wrap struct {
		Commands []struct {
			ID     string `json:"id"`
			Action string `json:"action"`
			Arg    string `json:"arg"`
		} `json:"commands"`
	}
	if json.Unmarshal(data, &wrap) != nil {
		return nil
	}
	for _, c := range wrap.Commands {
		if c.ID == "" || c.Action == "" {
			continue
		}
		res := runCmd(client, cfg, c.Action, c.Arg)
		_, _ = doJSON(client, http.MethodPost, cfg.Server+"/api/edge/cmd_result", cfg.Token, map[string]any{
			"device_id": cfg.DeviceID,
			"cmd_id":    c.ID,
			"action":    c.Action,
			"result":    res,
			"ts":        time.Now().Unix(),
		})
	}
	return nil
}

func runCmd(client *http.Client, cfg config, action, arg string) string {
	switch action {
	case "ping":
		return "pong"
	case "status", "metrics":
		b, _ := json.Marshal(collectMetrics())
		return string(b)
	case "reboot":
		if !strings.Contains(arg, "confirm=yes") {
			return "reboot: need arg confirm=yes"
		}
		go func() {
			time.Sleep(2 * time.Second)
			_ = exec.Command("reboot").Run()
		}()
		return "reboot scheduled"
	case "wifi_reload":
		out, _ := exec.Command("wifi", "reload").CombinedOutput()
		return truncate(string(out), 8000)
	case "network_reload":
		out, _ := exec.Command("/etc/init.d/network", "reload").CombinedOutput()
		return truncate(string(out), 8000)
	case "uci_get":
		out, _ := exec.Command("uci", "-q", "get", arg).CombinedOutput()
		return truncate(string(out), 8000)
	case "uci_show":
		args := []string{"-q", "show"}
		if arg != "" {
			args = append(args, arg)
		}
		out, _ := exec.Command("uci", args...).CombinedOutput()
		return truncate(string(out), 8000)
	case "uci_set":
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return "uci_set: need path=value"
		}
		out, err := exec.Command("uci", "set", parts[0]+"="+parts[1]).CombinedOutput()
		if err != nil {
			return truncate(string(out)+" "+err.Error(), 8000)
		}
		return "ok"
	case "uci_commit":
		out, _ := exec.Command("uci", "commit").CombinedOutput()
		return truncate(string(out), 4000)
	case "uci_batch":
		return uciBatch(arg)
	case "logread":
		out, _ := exec.Command("logread").CombinedOutput()
		lines := strings.Split(string(out), "\n")
		if len(lines) > 80 {
			lines = lines[len(lines)-80:]
		}
		return strings.Join(lines, "\n")
	case "config_backup":
		return configBackup(client, cfg)
	case "config_restore":
		return configRestore(client, cfg, arg)
	case "apply_template", "bootstrap_apply":
		return applyTemplate(client, cfg)
	case "agent_update":
		if !strings.Contains(arg, "confirm=yes") {
			return "agent_update: need confirm=yes in arg (URL|sha|confirm=yes)"
		}
		return agentUpdate(arg)
	case "sysupgrade":
		return doSysupgrade(arg)
	default:
		return "denied:" + action
	}
}

func uciBatch(arg string) string {
	var ok, fail int
	var logs []string
	for _, line := range strings.Split(arg, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "commit" {
			out, err := exec.Command("uci", "commit").CombinedOutput()
			logs = append(logs, "commit:"+truncate(string(out), 200))
			if err != nil {
				fail++
			} else {
				ok++
			}
			continue
		}
		if line == "network_reload" {
			out, _ := exec.Command("/etc/init.d/network", "reload").CombinedOutput()
			logs = append(logs, "network_reload:"+truncate(string(out), 200))
			ok++
			continue
		}
		if line == "wifi_reload" {
			out, _ := exec.Command("wifi", "reload").CombinedOutput()
			logs = append(logs, "wifi_reload:"+truncate(string(out), 200))
			ok++
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			logs = append(logs, "bad:"+line)
			fail++
			continue
		}
		out, err := exec.Command("uci", "set", parts[0]+"="+parts[1]).CombinedOutput()
		if err != nil {
			logs = append(logs, "fail:"+parts[0]+":"+truncate(string(out), 100))
			fail++
		} else {
			ok++
		}
	}
	return fmt.Sprintf("ok=%d fail=%d\n%s", ok, fail, strings.Join(logs, "\n"))
}

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
		return edgeagent.FormatApplyReport(toSet) + applyVPNClient(tmpl)
	}
	for _, line := range toSet {
		_ = exec.Command("uci", "set", line).Run()
	}
	_ = exec.Command("uci", "commit").Run()
	_ = exec.Command("/etc/init.d/network", "reload").Run()
	_ = exec.Command("wifi", "reload").Run()
	vpnNote := applyVPNClient(tmpl)
	return edgeagent.FormatApplyReport(toSet) + "\n" + strings.Join(toSet, "\n") + vpnNote
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

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
