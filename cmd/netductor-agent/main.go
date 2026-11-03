package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/guest"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// Follow internal/version.Release — never hardcode.
var version = ndver.Release

var mtlsRefreshInstalled bool
var mtlsRefreshBackupActive bool

type config struct {
	Server   string
	Token    string
	DeviceID string
	Interval int
	Role     string // edge (default) | build
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
	if strings.EqualFold(cfg.Role, "build") {
		runBuildAgent(cfg)
		return
	}
	client := agentHTTPClient()
	mtlsRefreshBackupActive = edgeMTLSBackupsPresent()
	startRecoveryHTTP(&cfg)
	startGuestHTTP(&cfg)

	// Offline-first: apply local UCI overlay even with no WAN.
	applyLocalOverlayOnce()

	backoff := time.Duration(cfg.Interval) * time.Second
	if backoff < 15*time.Second {
		backoff = 15 * time.Second
	}
	mtlsRefreshFailures := 0
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
		luciTickAutoDisable()
		checkRelayAndFallback()
		if err := heartbeat(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "heartbeat: %v\n", err)
			if mtlsRefreshBackupActive && !agentTLSHandshakeSucceeded.Load() {
				mtlsRefreshFailures++
				if mtlsRefreshFailures >= 3 && restoreEdgeMTLSMaterial() {
					fmt.Fprintln(os.Stderr, "mTLS refresh did not complete TLS handshake; restored previous client material")
					client.CloseIdleConnections()
					agentTLSHandshakeSucceeded.Store(false)
					client = agentHTTPClient()
					mtlsRefreshFailures = 0
				}
			}
		} else {
			mtlsRefreshFailures = 0
		}
		if err := pollCmds(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "poll: %v\n", err)
		}
		if mtlsRefreshInstalled {
			client.CloseIdleConnections()
			agentTLSHandshakeSucceeded.Store(false)
			client = agentHTTPClient()
			mtlsRefreshInstalled = false
			mtlsRefreshFailures = 0
		}
		time.Sleep(time.Duration(cfg.Interval) * time.Second)
	}
}

// applyLocalOverlayOnce applies /etc/netductor-agent/local.uci once (marker local_applied).
// Marker is written only after a successful uciBatch (fail=0).
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
	if uciBatchFailed(res) {
		fmt.Fprintf(os.Stderr, "local.uci: failed — marker not written, will retry\n")
		return
	}
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

func saveDeviceToken(tok string) error {
	if err := os.MkdirAll(agentDir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(deviceTokenPath(), []byte(tok+"\n"), 0o600)
}

func clearDeviceToken() {
	_ = os.Remove(deviceTokenPath())
	_ = os.Remove(tokenAckDonePath())
	_ = os.Remove(filepath.Join(agentDir(), "token_ack_pending")) // legacy
}

func tokenAckDonePath() string {
	return filepath.Join(agentDir(), "token_ack_done")
}

// tokenAckDone is true only when the marker exists and holds the full expected payload.
func tokenAckDone() bool {
	b, err := os.ReadFile(tokenAckDonePath())
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b)) == "ok"
}

func markTokenAckDone() error {
	if err := os.MkdirAll(agentDir(), 0o700); err != nil {
		return err
	}
	path := tokenAckDonePath()
	tmp := path + ".tmp"
	payload := []byte("ok\n")
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	got, err := os.ReadFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if string(got) != string(payload) {
		_ = os.Remove(tmp)
		return fmt.Errorf("token_ack_done temp incomplete")
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	got, err = os.ReadFile(path)
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	if strings.TrimSpace(string(got)) != "ok" {
		_ = os.Remove(path)
		return fmt.Errorf("token_ack_done verify failed")
	}
	return nil
}

// sendTokenAck posts confirmation that device_token was persisted. Returns nil only on HTTP 2xx.
func sendTokenAck(client *http.Client, server, deviceID, deviceToken string) error {
	ackBody, _ := json.Marshal(map[string]string{"device_id": deviceID})
	ackReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(server, "/")+"/api/edge/token-ack", bytes.NewReader(ackBody))
	if err != nil {
		return err
	}
	ackReq.Header.Set("Authorization", "Bearer "+deviceToken)
	ackReq.Header.Set("Content-Type", "application/json")
	ackResp, err := client.Do(ackReq)
	if err != nil {
		return err
	}
	defer ackResp.Body.Close()
	if ackResp.StatusCode >= 300 {
		b, _ := io.ReadAll(ackResp.Body)
		return fmt.Errorf("token-ack HTTP %d: %s", ackResp.StatusCode, truncate(string(b), 120))
	}
	return nil
}

// ensureTokenAcked retries token-ack until primary confirms. Marker is written atomically and
// verified; marker write errors are returned so the next cycle retries instead of skipping.
func ensureTokenAcked(client *http.Client, cfg *config, tok string) error {
	if tokenAckDone() {
		return nil
	}
	if err := sendTokenAck(client, cfg.Server, cfg.DeviceID, tok); err != nil {
		return err
	}
	if err := markTokenAckDone(); err != nil {
		return fmt.Errorf("token_ack_done: %w", err)
	}
	return nil
}

// ensureEnrolled uses bootstrap TOKEN until device_token is issued.
func ensureEnrolled(client *http.Client, cfg *config) error {
	if tok := loadDeviceToken(); tok != "" {
		cfg.Token = tok
		if err := ensureTokenAcked(client, cfg, tok); err != nil {
			return fmt.Errorf("token-ack: %w", err)
		}
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
			if err := saveDeviceToken(out.DeviceToken); err != nil {
				return fmt.Errorf("save device_token: %w", err)
			}
			cfg.Token = out.DeviceToken
			if err := ensureTokenAcked(client, cfg, out.DeviceToken); err != nil {
				return fmt.Errorf("token-ack: %w", err)
			}
			return nil
		}
		// Bootstrap enroll no longer returns device_token (R-05). Operator must install
		// token from Approve/Reissue, or recovery path must deliver it.
		return fmt.Errorf("approved without device_token — install token offline or use recovery enroll")
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
		case "ROLE":
			c.Role = strings.TrimSpace(v)
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

func countGuestActive() int {
	st, err := guest.OpenStore(guestStoreDir())
	if err != nil || st == nil {
		return -1
	}
	now := time.Now()
	n := 0
	for _, e := range st.Allow {
		if e.ExpiresAt.After(now) {
			n++
		}
	}
	return n
}

func countWifiClients() int {
	// iwinfo assoclist on all radios (OpenWrt)
	n := 0
	for _, phy := range []string{"wlan0", "wlan1", "phy0-ap0", "phy1-ap0", "rax0", "rax1"} {
		o, err := exec.Command("iwinfo", phy, "assoclist").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(o), "\n") {
			if strings.Contains(line, ":") && !strings.HasPrefix(strings.TrimSpace(line), "No") {
				// MAC lines look like "AA:BB:..."
				parts := strings.Fields(line)
				if len(parts) > 0 && strings.Count(parts[0], ":") == 5 {
					n++
				}
			}
		}
	}
	if n == 0 {
		return -1
	}
	return n
}

func measurePrimaryLatency() float64 {
	// NETDUCTOR_SERVER or config SERVER is https://host:8789
	server := strings.TrimSpace(os.Getenv("NETDUCTOR_SERVER"))
	if server == "" {
		if b, err := os.ReadFile("/etc/netductor-agent/config"); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "SERVER=") {
					server = strings.Trim(strings.TrimPrefix(line, "SERVER="), "\"'")
				}
			}
		}
	}
	if server == "" {
		return 0
	}
	u, err := url.Parse(server)
	if err != nil || u.Host == "" {
		return 0
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	start := time.Now()
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return 0
	}
	_ = c.Close()
	return float64(time.Since(start).Milliseconds())
}

func collectMetrics() map[string]any {
	m := map[string]any{
		"hostname": hostname(),
		"board":    boardName(),
		"agent":    version,
		"arch":     runtime.GOARCH,
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
	// Guest + LAN client counts (best-effort) for Sites topology cards.
	if n := countGuestActive(); n >= 0 {
		m["guest_active"] = n
	}
	if n := countWifiClients(); n >= 0 {
		m["device_count"] = n
	}
	// RTT to primary (from SERVER URL host) for latency tile.
	if ms := measurePrimaryLatency(); ms > 0 {
		m["latency_ms"] = ms
	}
	// Capability snapshot for primary UI card / plan (edge.DeviceFacts JSON).
	m["facts"] = collectDeviceFacts()
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
	data, err := doJSON(client, http.MethodPost, cfg.Server+"/api/edge/heartbeat", cfg.Token, payload)
	if err != nil {
		return err
	}
	var ack struct {
		MTLSRolloverConfirmed bool `json:"mtls_rollover_confirmed"`
		MTLSRotationConfirmed bool `json:"mtls_rotation_confirmed"`
		MTLSMaterialConfirmed bool `json:"mtls_material_confirmed"`
	}
	_ = json.Unmarshal(data, &ack)
	if ack.MTLSRolloverConfirmed || ack.MTLSRotationConfirmed || ack.MTLSMaterialConfirmed {
		cleanupEdgeMTLSBackups()
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
	if err := json.Unmarshal(data, &wrap); err != nil {
		return fmt.Errorf("commands json: %w", err)
	}
	var postErr error
	for _, c := range wrap.Commands {
		if c.ID == "" || c.Action == "" {
			continue
		}
		var res string
		if prev, ok := loadCmdDone(c.ID); ok {
			// Re-lease after lost result POST: do not re-execute side effects.
			res = prev
			if !strings.HasPrefix(res, "already:") {
				res = "already:" + res
			}
		} else if cmdStarted(c.ID) {
			// Crash between start and complete — refuse blind re-exec of side effects.
			res = "ambiguous: previous attempt started without completed marker — manual check required"
			if err := saveCmdDone(c.ID, res); err != nil {
				res = res + "; cmd_done save failed: " + err.Error()
				postErr = err
			}
		} else {
			if err := markCmdStarted(c.ID); err != nil {
				res = "cmd_done start marker: " + err.Error()
			} else {
				res = runCmd(client, cfg, c.Action, c.Arg)
				if err := saveCmdDone(c.ID, res); err != nil {
					res = res + "; cmd_done save failed: " + err.Error()
					postErr = err
				}
			}
		}
		if _, err := doJSON(client, http.MethodPost, cfg.Server+"/api/edge/cmd_result", cfg.Token, map[string]any{
			"device_id": cfg.DeviceID,
			"cmd_id":    c.ID,
			"action":    c.Action,
			"result":    res,
			"ts":        time.Now().Unix(),
		}); err != nil {
			// Keep server lease alive so the command can be re-delivered / result retried.
			postErr = err
		}
	}
	return postErr
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
		path, err := validUCIPath(arg)
		if err != nil {
			return "uci_get: invalid path"
		}
		out, _ := exec.Command("uci", "-q", "get", path).CombinedOutput()
		return truncate(string(out), 8000)
	case "uci_show":
		args := []string{"-q", "show"}
		if arg != "" {
			path, err := validUCIPath(arg)
			if err != nil {
				// allow package-only show: network, dhcp, …
				path, err = validUCIConfigName(arg)
				if err != nil {
					return "uci_show: invalid path"
				}
			}
			args = append(args, path)
		}
		out, _ := exec.Command("uci", args...).CombinedOutput()
		return truncate(string(out), 8000)
	case "uci_set":
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return "uci_set: need path=value"
		}
		path, err := validUCIPath(parts[0])
		if err != nil {
			return "uci_set: invalid path"
		}
		val, err := safeUCIValue(parts[1])
		if err != nil {
			return "uci_set: invalid value"
		}
		out, err := exec.Command("uci", "set", path+"="+val).CombinedOutput()
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
	case "storage_backup":
		return storageBackupB(client, cfg)
	case "storage_restore":
		return storageRestoreLocal(arg)
	case "storage_restore_remote":
		return storageRestoreRemote(client, cfg, arg)
	case "storage_init_usb":
		return storageInitUSB(arg)
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
	case "luci_enable", "luci_disable", "luci_extend", "luci_status":
		return luciCmd(action, arg)
	case "apply_template", "bootstrap_apply":
		return applyTemplate(client, cfg)
	case "mtls_refresh":
		return mtlsRefresh(client, cfg)
	case "apply_rsc":
		return "apply_rsc: not supported on OpenWrt agent (MikroTik RSC via sites/mikrotik path only)"
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
		CA     string `json:"ca_pem"`
		Cert   string `json:"cert_pem"`
		Key    string `json:"key_pem"`
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return "mtls_refresh: bad json"
	}
	if body.NodeID != cfg.DeviceID {
		return "mtls_refresh: returned node identity mismatch"
	}
	if err := validateEdgeMTLSMaterial(body.NodeID, body.CA, body.Cert, body.Key); err != nil {
		return "mtls_refresh: rejected material: " + err.Error()
	}
	if err := installEdgeMTLSMaterial(body.CA, body.Cert, body.Key); err != nil {
		return "mtls_refresh: install failed: " + err.Error()
	}
	mtlsRefreshInstalled = true
	mtlsRefreshBackupActive = true
	return "mtls material validated and staged; switching to new TLS identity"
}

func validateEdgeMTLSMaterial(nodeID, caPEM, certPEM, keyPEM string) error {
	if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(caPEM) == "" || strings.TrimSpace(certPEM) == "" || strings.TrimSpace(keyPEM) == "" {
		return fmt.Errorf("node ID, CA, certificate, and key are required")
	}
	pair, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return fmt.Errorf("certificate/key pair: %w", err)
	}
	if len(pair.Certificate) == 0 {
		return fmt.Errorf("client certificate is empty")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return fmt.Errorf("parse client certificate: %w", err)
	}
	if leaf.Subject.CommonName != "netductor-agent-"+nodeID {
		return fmt.Errorf("client certificate identity does not match device ID")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(caPEM)) {
		return fmt.Errorf("CA bundle contains no valid certificates")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return fmt.Errorf("client certificate chain: %w", err)
	}
	return nil
}

type edgeMTLSStage struct {
	dir     string
	stage   string
	backup  string
	swapped bool
}

func installEdgeMTLSMaterial(caPEM, certPEM, keyPEM string) error {
	stages := make([]edgeMTLSStage, 0, len(mtlsDirs))
	cleanupStages := func() {
		for _, stage := range stages {
			_ = os.RemoveAll(stage.stage)
		}
	}
	for _, dir := range mtlsDirs {
		parent := filepath.Dir(dir)
		if err := os.MkdirAll(parent, 0o700); err != nil {
			cleanupStages()
			return err
		}
		backup := dir + ".rollover-old"
		_, currentErr := os.Lstat(dir)
		backupInfo, backupErr := os.Lstat(backup)
		if backupErr == nil && (backupInfo.Mode()&os.ModeSymlink != 0 || !backupInfo.IsDir()) {
			cleanupStages()
			return fmt.Errorf("previous mTLS backup is not a real directory: %s", backup)
		}
		if backupErr == nil && os.IsNotExist(currentErr) {
			if err := os.Rename(backup, dir); err != nil {
				cleanupStages()
				return fmt.Errorf("recover previous mTLS directory %s: %w", dir, err)
			}
			currentErr = nil
		} else if backupErr == nil && currentErr == nil {
			if err := os.RemoveAll(backup); err != nil {
				cleanupStages()
				return fmt.Errorf("remove stale mTLS backup %s: %w", backup, err)
			}
		} else if backupErr != nil && !os.IsNotExist(backupErr) {
			cleanupStages()
			return fmt.Errorf("inspect previous mTLS backup %s: %w", backup, backupErr)
		}
		if currentErr != nil && !os.IsNotExist(currentErr) {
			cleanupStages()
			return fmt.Errorf("inspect current mTLS directory %s: %w", dir, currentErr)
		}
		if os.IsNotExist(currentErr) {
			if err := os.Mkdir(dir, 0o700); err != nil && !os.IsExist(err) {
				cleanupStages()
				return fmt.Errorf("create empty mTLS directory %s: %w", dir, err)
			}
		}
		stageDir, err := os.MkdirTemp(parent, ".mtls-staging-*")
		if err != nil {
			cleanupStages()
			return err
		}
		stages = append(stages, edgeMTLSStage{dir: dir, stage: stageDir, backup: backup})
		if err := copyAgentMTLSExtras(dir, stageDir, true); err != nil {
			cleanupStages()
			return fmt.Errorf("preserve mTLS directory %s: %w", dir, err)
		}
		for name, data := range map[string]string{
			"ca.crt": caPEM, "client.crt": certPEM, "client.key": keyPEM,
		} {
			if err := writeAgentFileDurable(filepath.Join(stageDir, name), []byte(data), 0o600); err != nil {
				cleanupStages()
				return fmt.Errorf("stage %s in %s: %w", name, dir, err)
			}
		}
		if err := syncAgentDirectory(stageDir); err != nil {
			cleanupStages()
			return fmt.Errorf("sync staged mTLS directory %s: %w", dir, err)
		}
	}
	rollback := func() error {
		var first error
		for i := len(stages) - 1; i >= 0; i-- {
			stage := &stages[i]
			if !stage.swapped {
				continue
			}
			if err := os.RemoveAll(stage.dir); err != nil && first == nil {
				first = err
			}
			if err := os.Rename(stage.backup, stage.dir); err != nil && first == nil {
				first = err
			}
			if err := syncAgentDirectory(filepath.Dir(stage.dir)); err != nil && first == nil {
				first = err
			}
		}
		return first
	}
	for i := range stages {
		stage := &stages[i]
		if err := os.Rename(stage.dir, stage.backup); err != nil {
			_ = rollback()
			cleanupStages()
			return fmt.Errorf("backup current mTLS directory %s: %w", stage.dir, err)
		}
		if err := os.Rename(stage.stage, stage.dir); err != nil {
			_ = os.Rename(stage.backup, stage.dir)
			_ = rollback()
			cleanupStages()
			return fmt.Errorf("activate mTLS directory %s: %w", stage.dir, err)
		}
		stage.swapped = true
		if err := syncAgentDirectory(filepath.Dir(stage.dir)); err != nil {
			_ = rollback()
			cleanupStages()
			return fmt.Errorf("sync activated mTLS directory %s: %w", stage.dir, err)
		}
	}
	return nil
}

func copyAgentMTLSExtras(source, destination string, root bool) error {
	info, err := os.Lstat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("mTLS source is not a real directory: %s", source)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if root && (name == "ca.crt" || name == "client.crt" || name == "client.key") {
			continue
		}
		src, dst := filepath.Join(source, name), filepath.Join(destination, name)
		info, err := os.Lstat(src)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in mTLS directory: %s", name)
		}
		if info.IsDir() {
			if err := os.Mkdir(dst, 0o700); err != nil && !os.IsExist(err) {
				return err
			}
			if err := copyAgentMTLSExtras(src, dst, false); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing non-regular mTLS entry: %s", name)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := writeAgentFileDurable(dst, data, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func writeAgentFileDurable(dst string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".mtls-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	n, err := tmp.Write(data)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if n != len(data) {
		_ = tmp.Close()
		return fmt.Errorf("short write: %d of %d bytes", n, len(data))
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	return syncAgentDirectory(dir)
}

func syncAgentDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func edgeMTLSBackupsPresent() bool {
	for _, dir := range mtlsDirs {
		info, err := os.Lstat(dir + ".rollover-old")
		if err == nil && info.Mode()&os.ModeSymlink == 0 && info.IsDir() {
			return true
		}
	}
	return false
}

func restoreEdgeMTLSMaterial() bool {
	for i := len(mtlsDirs) - 1; i >= 0; i-- {
		dir := mtlsDirs[i]
		backup := dir + ".rollover-old"
		backupInfo, err := os.Lstat(backup)
		if os.IsNotExist(err) {
			if _, currentErr := os.Lstat(dir); currentErr == nil {
				continue // this directory was already restored by a prior retry
			}
			return false
		}
		if err != nil || backupInfo.Mode()&os.ModeSymlink != 0 || !backupInfo.IsDir() {
			return false
		}
		failed := dir + ".rollover-failed"
		_ = os.RemoveAll(failed)
		if _, err := os.Lstat(dir); err == nil {
			if err := os.Rename(dir, failed); err != nil {
				return false
			}
		}
		if err := os.Rename(backup, dir); err != nil {
			_ = os.Rename(failed, dir)
			return false
		}
		_ = os.RemoveAll(failed)
		if err := syncAgentDirectory(filepath.Dir(dir)); err != nil {
			return false
		}
	}
	mtlsRefreshBackupActive = false
	return true
}

func cleanupEdgeMTLSBackups() {
	for _, dir := range mtlsDirs {
		_ = os.RemoveAll(dir + ".rollover-old")
		_ = syncAgentDirectory(filepath.Dir(dir))
	}
	mtlsRefreshBackupActive = false
}

