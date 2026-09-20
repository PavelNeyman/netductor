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
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/edgeagent"
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/tapo"
)

var version = "0.8.1"

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
		case "help", "-h", "--help":
			fmt.Print(`netductor-agent — outbound edge agent for OpenWrt

Config /etc/netductor-agent/config:
  SERVER=http://vps:8787
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
	cfg.Server = strings.TrimRight(cfg.Server, "/")
	client := &http.Client{Timeout: 120 * time.Second}
	startRecoveryHTTP(&cfg)

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

func nvrDir(cfg config) string {
	if cfg.NVRDir != "" {
		return cfg.NVRDir
	}
	return "/tmp/netductor-nvr"
}

func nvrMaxBytes(cfg config) int64 {
	mb := cfg.NVRMaxMB
	if mb <= 0 {
		mb = 24 // safe default for 128MB RAM tmpfs
	}
	if mb > 4096 {
		mb = 4096
	}
	return int64(mb) * 1024 * 1024
}

func nvrRecordStart(client *http.Client, cfg config, arg string) string {
	parts := strings.Split(arg, "|")
	if len(parts) < 2 {
		return "error:arg camera_id|rtsp_url[|segment_sec]"
	}
	camID := strings.TrimSpace(parts[0])
	url := strings.TrimSpace(parts[1])
	seg := 300
	if len(parts) >= 3 {
		if n, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil && n > 0 {
			seg = n
		}
	}
	if camID == "" || url == "" {
		return "error:empty"
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "error:ffmpeg not installed"
	}
	nvrRecMu.Lock()
	running := 0
	for id, c := range nvrRecCmds {
		if id != camID && c != nil && c.Process != nil {
			running++
		}
	}
	nvrRecMu.Unlock()
	if running >= nvrMaxConcurrent {
		return "error:max concurrent records (" + strconv.Itoa(nvrMaxConcurrent) + ")"
	}
	dir := filepath.Join(nvrDir(cfg), camID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "error:" + err.Error()
	}
	nvrRecordStop(camID)
	stop := make(chan struct{})
	nvrRecMu.Lock()
	nvrRecStop[camID] = stop
	nvrRecMu.Unlock()
	go nvrSupervise(client, cfg, camID, url, seg, dir, stop)
	// dir already under nvrDir(cfg)
	go nvrUploadLoop(client, cfg, camID, dir, stop)
	return "ok:recording:" + camID + " (supervised, max " + strconv.Itoa(nvrMaxConcurrent) + " cams)"
}

// nvrSupervise runs ffmpeg with backoff when RTSP is down (no tight restart loop).
func nvrSupervise(client *http.Client, cfg config, camID, url string, seg int, dir string, stop chan struct{}) {
	backoff := 5 * time.Second
	const maxBackoff = 5 * time.Minute
	for {
		select {
		case <-stop:
			return
		default:
		}
		// cheap TCP check before spawning ffmpeg
		if err := nvrTCPCheckRTSP(url); err != nil {
			fmt.Fprintf(os.Stderr, "nvr %s: offline %v; retry in %s\n", camID, err, backoff)
			select {
			case <-stop:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		backoff = 5 * time.Second
		_ = nvrTrimTmp(cfg)
		cmd := exec.Command("ffmpeg",
			"-hide_banner", "-loglevel", "error",
			"-rtsp_transport", "tcp",
			"-timeout", "5000000", // 5s in microseconds (some builds)
			"-rw_timeout", "5000000",
			"-stimeout", "5000000",
			"-i", url,
			"-c", "copy",
			"-f", "segment",
			"-segment_time", strconv.Itoa(seg),
			"-segment_atclocktime", "1",
			"-strftime", "1",
			"-reset_timestamps", "1",
			"-break_non_keyframes", "1",
			filepath.Join(dir, "%Y%m%d-%H%M%S.mp4"),
		)
		nvrRecMu.Lock()
		nvrRecCmds[camID] = cmd
		nvrRecMu.Unlock()
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "nvr %s: start %v\n", camID, err)
			select {
			case <-stop:
				return
			case <-time.After(backoff):
			}
			continue
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-stop:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
			nvrRecMu.Lock()
			delete(nvrRecCmds, camID)
			nvrRecMu.Unlock()
			return
		case err := <-done:
			nvrRecMu.Lock()
			if nvrRecCmds[camID] == cmd {
				delete(nvrRecCmds, camID)
			}
			nvrRecMu.Unlock()
			fmt.Fprintf(os.Stderr, "nvr %s: ffmpeg exited %v; retry in %s\n", camID, err, backoff)
			select {
			case <-stop:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func nvrTCPCheckRTSP(url string) error {
	hostport := url
	if strings.HasPrefix(url, "rtsp://") {
		u := url[7:]
		if i := strings.Index(u, "@"); i >= 0 {
			u = u[i+1:]
		}
		if i := strings.IndexAny(u, "/?"); i >= 0 {
			u = u[:i]
		}
		hostport = u
	}
	if !strings.Contains(hostport, ":") {
		hostport += ":554"
	}
	c, err := net.DialTimeout("tcp", hostport, 3*time.Second)
	if err != nil {
		return err
	}
	_ = c.Close()
	return nil
}

func nvrTrimTmp(cfg config) error {
	root := nvrDir(cfg)
	limit := nvrMaxBytes(cfg)
	var files []struct {
		path string
		mod  time.Time
		size int64
	}
	var total int64
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		files = append(files, struct {
			path string
			mod  time.Time
			size int64
		}{path, info.ModTime(), info.Size()})
		return nil
	})
	if total <= limit {
		return nil
	}
	// oldest first
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].mod.Before(files[i].mod) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	for _, f := range files {
		if total <= limit*8/10 {
			break
		}
		_ = os.Remove(f.path)
		total -= f.size
	}
	return nil
}




func cameraPTZ(arg string) string {
	// Preferred: native Go port of pytapo (HA Tapo-Control protocol).
	// Fallback: python scripts/tapo_control.py, then ONVIF :2020.
	parts := strings.Split(arg, "|")
	if len(parts) < 4 {
		return "error:arg ip|user|pass|left|right|up|down|stop|night:auto|privacy:off"
	}
	ip, user, pass, dir := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), parts[2], strings.TrimSpace(parts[3])
	step := 10
	if len(parts) >= 5 {
		if n, err := strconv.Atoi(strings.TrimSpace(parts[4])); err == nil {
			step = n
		}
	}
	out := tapo.Control(ip, user, pass, dir, step)
	if strings.HasPrefix(out, "tapo-go:") && !strings.Contains(out, "tapo-go:login:") && !strings.Contains(out, "tapo-go:err:") {
		return out
	}
	// if login failed, still try python helper / ONVIF
	if pyOut, ok := tryTapoControlPy(ip, user, pass, dir, step); ok {
		return pyOut + " | first=" + out
	}
	if strings.HasPrefix(dir, "night:") || strings.HasPrefix(dir, "privacy:") {
		return out + " | need working tapo-go or pytapo"
	}
	onv := nvr.ONVIFPTZ(ip, user, pass, dir, 800)
	return onv + " | first=" + out
}

func tryTapoControlPy(ip, user, pass, dir string, step int) (string, bool) {
	py, err := exec.LookPath("python3")
	if err != nil {
		py, err = exec.LookPath("python")
		if err != nil {
			return "", false
		}
	}
	script := "/opt/netductor/scripts/tapo_control.py"
	if _, err := os.Stat(script); err != nil {
		script = "/usr/share/netductor/tapo_control.py"
	}
	if _, err := os.Stat(script); err != nil {
		// try next to agent binary
		if exe, e := os.Executable(); e == nil {
			cand := filepath.Join(filepath.Dir(exe), "tapo_control.py")
			if _, err := os.Stat(cand); err == nil {
				script = cand
			}
		}
	}
	if _, err := os.Stat(script); err != nil {
		return "", false
	}
	var args []string
	if strings.HasPrefix(dir, "night:") {
		mode := strings.TrimPrefix(dir, "night:")
		args = []string{script, ip, user, pass, "night", mode}
	} else if strings.HasPrefix(dir, "privacy:") {
		mode := strings.TrimPrefix(dir, "privacy:")
		args = []string{script, ip, user, pass, "privacy", mode}
	} else {
		args = []string{script, ip, user, pass, "move", dir, strconv.Itoa(step)}
	}
	cmd := exec.Command(py, args...)
	b, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(b))
	if err != nil {
		return "pytapo:" + out + " err:" + err.Error(), true
	}
	return "pytapo:" + out, true
}

func nvrDiskInfo(cfg config) string {
	dir := nvrDir(cfg)
	_ = os.MkdirAll(dir, 0o700)
	out, err := exec.Command("df", "-h", dir).CombinedOutput()
	info := map[string]any{"dir": dir, "max_mb": cfg.NVRMaxMB, "df": strings.TrimSpace(string(out))}
	if err != nil {
		info["error"] = err.Error()
	}
	// rough used
	var used int64
	_ = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err == nil && fi != nil && !fi.IsDir() {
			used += fi.Size()
		}
		return nil
	})
	info["used_bytes"] = used
	info["limit_bytes"] = nvrMaxBytes(cfg)
	b, _ := json.Marshal(info)
	return string(b)
}

func nvrRecordStatus() string {
	nvrRecMu.Lock()
	defer nvrRecMu.Unlock()
	var ids []string
	for id, c := range nvrRecCmds {
		if c != nil && c.Process != nil {
			ids = append(ids, id)
		}
	}
	b, _ := json.Marshal(map[string]any{"active": ids, "count": len(ids), "max": nvrMaxConcurrent})
	return string(b)
}

func nvrRecordStop(camID string) string {
	camID = strings.TrimSpace(camID)
	nvrRecMu.Lock()
	if ch, ok := nvrRecStop[camID]; ok {
		select {
		case <-ch:
		default:
			close(ch)
		}
		delete(nvrRecStop, camID)
	}
	cmd := nvrRecCmds[camID]
	delete(nvrRecCmds, camID)
	nvrRecMu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		return "ok:stopped:" + camID
	}
	return "ok:not_running:" + camID
}

func nvrUploadLoop(client *http.Client, cfg config, camID, dir string, stop chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			nvrUploadDirOnce(client, cfg, camID, dir)
			return
		case <-ticker.C:
			nvrUploadDirOnce(client, cfg, camID, dir)
			_ = nvrTrimTmp(cfg)
		}
	}
}

func nvrUploadDirOnce(client *http.Client, cfg config, camID, dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)
		if !strings.HasSuffix(low, ".mp4") && !strings.HasSuffix(low, ".mkv") && !strings.HasSuffix(low, ".ts") {
			continue
		}
		path := filepath.Join(dir, name)
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(fi.ModTime()) < 8*time.Second {
			continue
		}
		if err := nvrUploadFile(client, cfg, camID, path); err == nil {
			_ = os.Remove(path)
		}
	}
}

func nvrUploadFile(client *http.Client, cfg config, camID, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("camera_id", camID)
	part, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	_ = w.Close()
	req, err := http.NewRequest(http.MethodPost, cfg.Server+"/api/nvr/ingest", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	tok := loadDeviceToken()
	if tok == "" {
		tok = cfg.Token
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("ingest %d %s", resp.StatusCode, string(b))
	}
	return nil
}


func rtspProbe(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "error:empty"
	}
	// Accept full rtsp URL or host:port
	hostport := arg
	path := ""
	if strings.HasPrefix(arg, "rtsp://") {
		// rtsp://user:pass@host:port/path
		u := arg[7:]
		if i := strings.Index(u, "@"); i >= 0 {
			u = u[i+1:]
		}
		if i := strings.IndexAny(u, "/?"); i >= 0 {
			path = u[i:]
			u = u[:i]
		}
		hostport = u
	}
	if !strings.Contains(hostport, ":") {
		hostport = hostport + ":554"
	}
	conn, err := net.DialTimeout("tcp", hostport, 5*time.Second)
	if err != nil {
		return "error:tcp:" + err.Error()
	}
	_ = conn.Close()
	out := "ok:tcp:" + hostport
	if path != "" {
		out += " path=" + path
	}
	// optional ffprobe if present (no password echo)
	if _, err := exec.LookPath("ffprobe"); err == nil && strings.HasPrefix(arg, "rtsp://") {
		cmd := exec.Command("ffprobe", "-v", "error", "-rtsp_transport", "tcp",
			"-timeout", "5000000", "-rw_timeout", "5000000",
			"-show_entries", "stream=codec_type", "-of", "csv=p=0", arg)
		cmd.Stdout = nil
		done := make(chan struct{})
		var b []byte
		var err error
		go func() {
			b, err = cmd.CombinedOutput()
			close(done)
		}()
		select {
		case <-done:
			if err != nil {
				out += "; ffprobe:fail:" + truncate(string(b), 120)
			} else {
				out += "; ffprobe:ok:" + truncate(strings.TrimSpace(string(b)), 80)
			}
		case <-time.After(8 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			out += "; ffprobe:timeout"
		}
	}
	return out
}


func dhcpLeasesJSON() string {
	b, err := os.ReadFile("/tmp/dhcp.leases")
	if err != nil {
		// try dnsmasq alternate
		b, err = os.ReadFile("/var/dhcp.leases")
		if err != nil {
			return "error:" + err.Error()
		}
	}
	type lease struct {
		Expiry   int64  `json:"expiry"`
		MAC      string `json:"mac"`
		IP       string `json:"ip"`
		Hostname string `json:"hostname,omitempty"`
		ClientID string `json:"client_id,omitempty"`
	}
	var out []lease
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		exp, _ := strconv.ParseInt(f[0], 10, 64)
		l := lease{Expiry: exp, MAC: strings.ToLower(strings.ReplaceAll(f[1], "-", ":")), IP: f[2]}
		if len(f) >= 4 && f[3] != "*" {
			l.Hostname = f[3]
		}
		if len(f) >= 5 && f[4] != "*" {
			l.ClientID = f[4]
		}
		out = append(out, l)
	}
	raw, _ := json.Marshal(map[string]any{"leases": out, "count": len(out)})
	return string(raw)
}

func wifiClientsJSON() string {
	// Best-effort: iwinfo <if> assoclist for each radio iface
	ifaces := []string{}
	if out, err := exec.Command("iwinfo").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// "wlan0     ESSID: ..."
			parts := strings.Fields(line)
			if len(parts) > 0 && !strings.Contains(parts[0], ":") {
				ifaces = append(ifaces, parts[0])
			}
		}
	}
	type client struct {
		Iface string `json:"iface"`
		MAC   string `json:"mac"`
		Raw   string `json:"raw,omitempty"`
	}
	var clients []client
	seen := map[string]bool{}
	for _, ifc := range ifaces {
		if seen[ifc] {
			continue
		}
		seen[ifc] = true
		out, err := exec.Command("iwinfo", ifc, "assoclist").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			mac := strings.ToLower(strings.ReplaceAll(fields[0], "-", ":"))
			if strings.Count(mac, ":") != 5 {
				continue
			}
			clients = append(clients, client{Iface: ifc, MAC: mac, Raw: truncate(line, 160)})
		}
	}
	raw, _ := json.Marshal(map[string]any{"clients": clients, "count": len(clients)})
	return string(raw)
}

// dhcpStaticHost arg: mac=aa:bb:..|ip=192.168.1.50|name=tapo1
func dhcpStaticHost(arg string) string {
	kv := map[string]string{}
	for _, p := range strings.Split(arg, "|") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		i := strings.IndexByte(p, '=')
		if i <= 0 {
			continue
		}
		kv[strings.ToLower(p[:i])] = p[i+1:]
	}
	mac := strings.ToLower(strings.ReplaceAll(kv["mac"], "-", ":"))
	ip := kv["ip"]
	name := kv["name"]
	if mac == "" || ip == "" {
		return "error:need mac= and ip="
	}
	if name == "" {
		name = "cam-" + strings.ReplaceAll(mac, ":", "")[8:]
	}
	// Find existing host section with same mac or add new
	show, _ := exec.Command("uci", "-q", "show", "dhcp").CombinedOutput()
	section := ""
	for _, line := range strings.Split(string(show), "\n") {
		if strings.Contains(line, ".mac=") && strings.Contains(strings.ToLower(line), mac) {
			// dhcp.@host[N].mac='...'
			left := strings.SplitN(line, ".mac=", 2)[0]
			section = left
			break
		}
	}
	var lines []string
	if section == "" {
		lines = append(lines, "add dhcp host")
		section = "dhcp.@host[-1]"
	}
	lines = append(lines,
		"set "+section+".mac="+mac,
		"set "+section+".ip="+ip,
		"set "+section+".name="+name,
		"set "+section+".dns=1",
		"commit dhcp",
	)
	res := uciBatch(strings.Join(lines, "\n"))
	// reload dnsmasq
	out, _ := exec.Command("/etc/init.d/dnsmasq", "reload").CombinedOutput()
	return res + "; dnsmasq:" + truncate(string(out), 120)
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
