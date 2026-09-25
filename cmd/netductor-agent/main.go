package main

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var version = "0.9.14"

type config struct {
	Server   string
	Token    string
	DeviceID string
	Interval int
	NVRDir   string // buffer path; default /tmp/netductor-nvr (tmpfs)
	NVRMaxMB int    // max buffer size MB; default 24 on tmpfs
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Printf("netductor-agent %s\n", version)
			return
		case "guest":
			runGuestCLI(os.Args[2:])
			return
		case "help", "-h", "--help":
			fmt.Print(`netductor-agent — outbound edge agent for OpenWrt

Config /etc/netductor-agent/config:
  SERVER=https://vps:8789
  TOKEN=...
  DEVICE_ID=site1
  INTERVAL=60
  NVR_DIR=/tmp/netductor-nvr   # USB e.g. /mnt/sda1/netductor-nvr
  NVR_MAX_MB=24                # raise on USB (e.g. 512)

Commands (from VPS):
  ping, status, metrics
  config_backup          — tar /etc/config → VPS
  uci_get|show|set|commit|batch
  wifi_reload, network_reload, reboot
  agent_update           — arg: URL or URL|sha256
  guest                 — guest Wi‑Fi desk/captive (see guest -h)
  sysupgrade             — arg: URL|sha256|confirm=yes
`)
			return
		}
	}
	cfg := loadConfig()
	if cfg.Server == "" || cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "SERVER and TOKEN required in /etc/netductor-agent/config")
		os.Exit(1)
	}
	cfg.Server = normalizeServerURL(cfg.Server)
	client := agentHTTPClient()
	startRecoveryHTTP(&cfg)
	startGuestHTTP(&cfg)

	// Offline-first: apply local UCI overlay even with no WAN.
	applyLocalOverlayOnce()

	backoff := time.Duration(cfg.Interval) * time.Second
	if backoff < 15*time.Second {
		backoff = 15 * time.Second
	}
	for {
		if err := ensureEnrolled(client, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "enroll: %v (retry in %s)\n", err, backoff)
			time.Sleep(backoff)
			if strings.Contains(err.Error(), "pending") {
				backoff = time.Duration(cfg.Interval) * time.Second
			} else if backoff < 5*time.Minute {
				backoff *= 2
				if backoff > 5*time.Minute {
					backoff = 5 * time.Minute
				}
			}
			continue
		}
		backoff = time.Duration(cfg.Interval) * time.Second

		if os.Getenv("CONTROL_ONLY") == "1" || readConfigFlag("CONTROL_ONLY") == "1" {
			// recovery / control-plane only — do not push site templates
		} else if _, err := os.Stat(filepath.Join(agentDir(), "applied_template")); err != nil {
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

// applyLocalOverlayOnce applies /etc/netductor-agent/local.uci once (marker local_applied).
func applyLocalOverlayOnce() {
	marker := filepath.Join(agentDir(), "local_applied")
	if _, err := os.Stat(marker); err == nil {
		return
	}
	path := filepath.Join(agentDir(), "local.uci")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	res := uciBatch(string(b))
	fmt.Fprintf(os.Stderr, "local.uci: %s\n", res)
	_ = os.WriteFile(marker, []byte(res+"\n"), 0o600)
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
	if v := os.Getenv("NETDUCTOR_NVR_DIR"); v != "" {
		c.NVRDir = v
	}
	if v := os.Getenv("NETDUCTOR_NVR_MAX_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.NVRMaxMB = n
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
		case "NVR_DIR":
			c.NVRDir = strings.TrimSpace(v)
		case "NVR_MAX_MB":
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				c.NVRMaxMB = n
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
	relayHost := strings.TrimSpace(readFirstLine("/etc/netductor-agent/secondary.host"))
	if relayHost == "" {
		return
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	c, err := d.Dial("tcp", net.JoinHostPort(relayHost, "443"))
	ok := err == nil
	if ok {
		_ = c.Close()
	}
	statePath := "/etc/netductor-agent/secondary.health"
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
	if ser := localClientCertSerial(); ser != "" {
		payload["cert_serial"] = ser
	}
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
	case "guest_status":
		return guestCmdStatus()
	case "guest_grant":
		return guestCmdGrant(arg)
	case "guest_revoke":
		return guestCmdRevoke(arg)
	case "guest_apply_template":
		return applyTemplate(client, cfg)
	case "apply_template", "bootstrap_apply":
		return applyTemplate(client, cfg)
	case "mtls_refresh":
		return mtlsRefresh(client, cfg)
	case "agent_update":
		if !strings.Contains(arg, "confirm=yes") {
			return "agent_update: need confirm=yes in arg (URL|sha|confirm=yes)"
		}
		return agentUpdate(arg)
	case "sysupgrade":
		return doSysupgrade(arg)
	case "dhcp_leases":
		return dhcpLeasesJSON()
	case "wifi_clients":
		return wifiClientsJSON()
	case "dhcp_static":
		return dhcpStaticHost(arg)
	case "rtsp_probe":
		return rtspProbe(arg)
	case "nvr_record_start":
		return nvrRecordStart(client, cfg, arg)
	case "nvr_record_stop":
		return nvrRecordStop(arg)
	case "nvr_record_status":
		return nvrRecordStatus()
	case "nvr_disk_info":
		return nvrDiskInfo(cfg)
	case "camera_ptz":
		return cameraPTZ(arg)
	default:
		return "denied:" + action
	}
}

var (
	nvrRecMu   sync.Mutex
	nvrRecCmds = map[string]*exec.Cmd{}
	nvrRecStop = map[string]chan struct{}{}
)

const nvrMaxConcurrent = 2

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func readConfigFlag(key string) string {
	b, err := os.ReadFile(filepath.Join(agentDir(), "config"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimSpace(strings.TrimPrefix(line, key+"="))
		}
	}
	return ""
}

func localClientCertSerial() string {
	for _, p := range []string{
		"/etc/netductor-agent/mtls/client.crt",
		"/etc/netductor/secrets/mtls/client.crt",
	} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		block, _ := pem.Decode(b)
		if block == nil {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		return strings.ToLower(c.SerialNumber.Text(16))
	}
	return ""
}

func mtlsRefresh(client *http.Client, cfg config) string {
	data, err := doJSON(client, http.MethodGet, cfg.Server+"/api/edge/mtls/material", cfg.Token, nil)
	if err != nil {
		return "mtls_refresh: " + err.Error()
	}
	var body struct {
		CA   string `json:"ca_pem"`
		Cert string `json:"cert_pem"`
		Key  string `json:"key_pem"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return "mtls_refresh: bad json"
	}
	if body.Cert == "" || body.Key == "" {
		return "mtls_refresh: empty material"
	}
	dir := "/etc/netductor-agent/mtls"
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "ca.crt"), []byte(body.CA), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "client.crt"), []byte(body.Cert), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "client.key"), []byte(body.Key), 0o600)
	// also default path used by some installs
	_ = os.MkdirAll("/etc/netductor/secrets/mtls", 0o700)
	_ = os.WriteFile("/etc/netductor/secrets/mtls/ca.crt", []byte(body.CA), 0o600)
	_ = os.WriteFile("/etc/netductor/secrets/mtls/client.crt", []byte(body.Cert), 0o600)
	_ = os.WriteFile("/etc/netductor/secrets/mtls/client.key", []byte(body.Key), 0o600)
	return "mtls material written — restart agent to use new cert"
}
