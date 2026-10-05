package secondary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/svcpaths"
	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// AgentLoop runs on RU VPS: heartbeat + pull config when version drifts.
func agentHTTPClient() *http.Client {
	c := &http.Client{Timeout: 20 * time.Second}
	if mtls.ClientReady() {
		if tlsCfg, err := mtls.ClientTLSConfig(); err == nil {
			c.Transport = &http.Transport{TLSClientConfig: tlsCfg}
		}
	}
	return c
}

var uplinkFailStreak int

func probePrimaryUplink(coreBase string) bool {
	hostport := strings.TrimPrefix(strings.TrimPrefix(coreBase, "https://"), "http://")
	host := hostport
	if i := strings.LastIndex(hostport, ":"); i > 0 {
		host = hostport[:i]
	}
	if host == "" {
		return false
	}
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, "443"), 5*time.Second)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func AgentLoop(coreBase, token string, interval time.Duration) {
	go StartRecoveryServer()
	if interval < 10*time.Second {
		interval = 30 * time.Second
	}
	client := agentHTTPClient()
	if mtls.ClientReady() {
		// Always use mTLS agent port; never plain :8788 when client certs exist.
		hostport := strings.TrimPrefix(strings.TrimPrefix(coreBase, "https://"), "http://")
		host := hostport
		if i := strings.LastIndex(hostport, ":"); i > 0 {
			host = hostport[:i]
		}
		coreBase = "https://" + host + ":" + mtls.AgentTLSPort
	}
	applied := 0
	var lastDone string
	var lastOK bool
	var lastLog string
	for {
		if err := agentTick(client, coreBase, token, &applied, &lastDone, &lastOK, &lastLog); err != nil {
			fmt.Fprintf(os.Stderr, "agent tick: %v\n", err)
		}
		time.Sleep(interval)
	}
}

func agentTick(client *http.Client, coreBase, token string, applied *int, lastDone *string, lastOK *bool, lastLog *string) error {
	// Re-read preferred core URL each tick (failover may switch tunnel↔public)
	if b, err := os.ReadFile("/etc/netductor/secrets/secondary_core_url"); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			coreBase = s
		}
	}
	// Service-plane health + failover state machine + apply (user uplink / agent URL)
	if _, sum, err := svcpaths.RunSecondaryCycle(); err != nil {
		fmt.Fprintf(os.Stderr, "svc-paths cycle: %v\n", err)
	} else if sum != "" && sum != "failover disabled" {
		// log only on action changes — Tick sets last_action
		st := svcpaths.LoadState()
		if st.LastAction != "" && st.LastAction != "none" {
			fmt.Fprintf(os.Stderr, "svc-paths: %s\n", sum)
		}
	}
	coreBase = strings.TrimRight(coreBase, "/")
	if mtls.ClientReady() {
		hostport := strings.TrimPrefix(strings.TrimPrefix(coreBase, "https://"), "http://")
		host := hostport
		if i := strings.LastIndex(hostport, ":"); i > 0 {
			host = hostport[:i]
		}
		coreBase = "https://" + host + ":" + mtls.AgentTLSPort
	}
	pub := readSecret("singbox_reality_public")
	sid := readSecret("singbox_short_id")
	sni := strings.TrimSpace(readFile(filepath.Join(paths.EtcDir(), "secrets", "singbox_reality_sni")))
	if sni == "" {
		sni = "ya.ru"
	}
	ip := publicIP()
	sbOK := false
	if out, err := exec.Command("systemctl", "is-active", "sing-box").Output(); err == nil {
		sbOK = strings.TrimSpace(string(out)) == "active"
	}
	// Probe public primary:443 (VLESS Reality path), not agent mTLS host
	pubHost := ""
	if b, err := os.ReadFile("/etc/netductor/secrets/secondary_core_url.public"); err == nil {
		s := strings.TrimSpace(string(b))
		s = strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
		if i := strings.IndexAny(s, ":/"); i > 0 {
			s = s[:i]
		}
		pubHost = s
	}
	upOK := false
	if pubHost != "" {
		upOK = probePrimaryUplink("https://" + pubHost + ":8789")
	} else {
		upOK = probePrimaryUplink(coreBase)
	}
	if upOK {
		uplinkFailStreak = 0
	} else {
		uplinkFailStreak++
		fmt.Fprintf(os.Stderr, "uplink probe primary:443 fail streak=%d\n", uplinkFailStreak)
		// after 3 consecutive fails: let svc-paths apply switch path; also restart mux if still on vless
		if uplinkFailStreak >= 3 && sbOK {
			st := svcpaths.LoadState()
			if st.UserPath == "vless" {
				fmt.Fprintf(os.Stderr, "uplink watchdog: restarting sing-box\n")
				_ = exec.Command("systemctl", "restart", "sing-box").Run()
			}
			uplinkFailStreak = 0
		}
	}
	cpu, memU, memT, load1 := sampleMetrics()
	mm := vpn.CollectMismatch(30)
	body, _ := json.Marshal(HeartbeatIn{
		PublicIP: ip, PBK: pub, SID: sid, SNI: sni,
		Version: version.Release, SingBoxOK: sbOK, UplinkOK: upOK, ConfigVer: *applied,
		CPUPercent: cpu, MemUsedMB: memU, MemTotalMB: memT, Load1: load1,
		CmdDone: *lastDone, CmdOK: *lastOK, CmdLog: *lastLog,
		MismatchTotal: mm.Total, MismatchByIP: mm.ByIP,
	})
	*lastDone, *lastOK, *lastLog = "", false, ""
	req, err := http.NewRequest(http.MethodPost, coreBase+"/api/secondary/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("heartbeat %s: %s", resp.Status, string(raw))
	}
	var hr struct {
		ConfigVer       int             `json:"config_ver"`
		NeedSync        bool            `json:"need_sync"`
		DesiredHostname string          `json:"desired_hostname"`
		DesiredRelease  string          `json:"desired_release"`
		Commands        []string        `json:"commands"`
		FailoverPolicy  json.RawMessage `json:"failover_policy"`
	}
	_ = json.Unmarshal(raw, &hr)
	if hn := strings.TrimSpace(hr.DesiredHostname); hn != "" {
		_ = applyHostname(hn)
	}
	if len(hr.FailoverPolicy) > 2 {
		_ = applyFailoverPolicy(hr.FailoverPolicy)
	}
	// desired_release is informational only — never auto-upgrade.
	// Version changes only via explicit agent command (upgrade / upgrade:vX) from operator.
	_ = hr.DesiredRelease
	for _, c := range hr.Commands {
		ok, log := runAgentCmd(c)
		*lastDone, *lastOK, *lastLog = c, ok, log
		// report immediately so core can TG-notify without waiting next tick
		_ = reportCmdDone(client, coreBase, token, c, ok, log, ip, pub, sid, sni, sbOK, *applied, cpu, memU, memT, load1)
		*lastDone, *lastOK, *lastLog = "", false, ""
	}
	if hr.NeedSync || hr.ConfigVer > *applied {
		if err := pullAndApply(client, coreBase, token); err != nil {
			return err
		}
		*applied = hr.ConfigVer
	}
	return nil
}

func pullAndApply(client *http.Client, coreBase, token string) error {
	req, err := http.NewRequest(http.MethodGet, coreBase+"/api/secondary/agent/config", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("config %s: %s", resp.Status, string(raw))
	}
	var b vpn.SecondaryBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return err
	}
	priv := readSecret("singbox_reality_private")
	sid := readSecret("singbox_short_id")
	if err := vpn.WriteSecondarySingBox(&b, priv, sid); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "restart", "sing-box").Run()
	_ = os.WriteFile(filepath.Join(paths.SecondaryDir(), "bundle.json"), raw, 0o600)
	return nil
}

func readSecret(name string) string {
	return paths.ReadSecret(name)
}
func readFile(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}
func publicIP() string {
	if v := os.Getenv("PUBLIC_IP"); v != "" {
		return v
	}
	out, err := exec.Command("curl", "-4", "-fsS", "--max-time", "5", "https://ifconfig.me").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sampleMetrics() (cpu float64, memUsed, memTotal int64, load1 float64) {
	// loadavg
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		fmt.Sscanf(string(b), "%f", &load1)
	}
	// meminfo
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail int64
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d", &total)
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(line, "MemAvailable: %d", &avail)
			}
		}
		memTotal = total / 1024
		if total > 0 {
			memUsed = (total - avail) / 1024
		}
	}
	cpu = load1 * 50 // rough indicator on small VPS
	if cpu > 100 {
		cpu = 100
	}
	return
}

func applyHostname(hn string) error {
	hn = strings.TrimSpace(hn)
	if hn == "" {
		return nil
	}
	_ = exec.Command("hostnamectl", "set-hostname", hn).Run()
	_ = os.WriteFile("/etc/hostname", []byte(hn+"\n"), 0o644)
	return nil
}

func reportCmdDone(client *http.Client, coreBase, token, cmd string, ok bool, log, ip, pub, sid, sni string, sbOK bool, applied int, cpu float64, memU, memT int64, load1 float64) error {
	upOK := probePrimaryUplink(coreBase)
	body, _ := json.Marshal(HeartbeatIn{
		PublicIP: ip, PBK: pub, SID: sid, SNI: sni,
		Version: version.Release, SingBoxOK: sbOK, UplinkOK: upOK, ConfigVer: applied,
		CPUPercent: cpu, MemUsedMB: memU, MemTotalMB: memT, Load1: load1,
		CmdDone: cmd, CmdOK: ok, CmdLog: log,
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(coreBase, "/")+"/api/secondary/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func runAgentCmd(cmd string) (ok bool, log string) {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "reboot":
		go func() {
			time.Sleep(2 * time.Second)
			_ = exec.Command("systemctl", "reboot").Run()
		}()
		return true, "reboot scheduled in 2s"
	case "upgrade":
		return secondaryUpgrade("")
	case "mtls_refresh":
		return secondaryMTLSRefresh()
	case "backup_pull":
		return secondaryBackupPull()
	case "metrics":
		return true, "metrics on next heartbeat"
	case "journal":
		out, err := exec.Command("journalctl", "-u", "sing-box", "-u", "netductor-secondary-agent", "-n", "60", "--no-pager", "-o", "short-iso").CombinedOutput()
		return err == nil || len(out) > 0, string(out)
	default:
		if strings.HasPrefix(cmd, "upgrade:") {
			return secondaryUpgrade(strings.TrimPrefix(cmd, "upgrade:"))
		}
		if cmd == "backup_local" {
			path, err := secondaryBackupLocal()
			if err != nil {
				return false, err.Error()
			}
			return true, path
		}
		if strings.HasPrefix(cmd, "restart:") {
			unit := strings.TrimPrefix(cmd, "restart:")
			// allowlist only netductor-related units
			allowed := map[string]bool{
				"sing-box":                  true,
				"netductor-secondary-agent": true,
				"netductor-api":             true,
				"netductor-telegram-bot":    true,
				"blocky":                    true,
			}
			if !allowed[unit] {
				return false, "restart denied: unit not in allowlist: " + unit
			}
			out, err := exec.Command("systemctl", "restart", unit).CombinedOutput()
			return err == nil, string(out)
		}
		return false, "unknown cmd: " + cmd
	}
}

func VersionHint() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_VERSION")); v != "" {
		return strings.TrimPrefix(v, "v")
	}
	if b, err := os.ReadFile("/etc/netductor/VERSION"); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return version.Release
}

// secondaryUpgrade downloads node+agent from GitHub Release (no apt storm).
// tag empty → VersionHint / latest file VERSION.
func secondaryUpgrade(tag string) (bool, string) {
	var log strings.Builder
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "v")
	if tag == "" {
		tag = VersionHint()
	}
	if tag == "" {
		tag = version.Release
	}
	tag = strings.TrimPrefix(tag, "v")
	for _, r := range tag {
		if (r < '0' || r > '9') && r != '.' {
			return false, "upgrade denied: tag must be a version"
		}
	}
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		arch = "amd64"
	}
	base := "https://github.com/PavelNeyman/netductor/releases/download/v" + tag + "/"
	log.WriteString(fmt.Sprintf("upgrade tag=v%s arch=%s\n", tag, arch))
	fetch := func(asset, dest string) error {
		url := base + asset
		tmp := "/tmp/" + asset + ".new"
		_ = os.Remove(tmp)
		// curl: show HTTP code on failure (wget -q hides reason → bare "exit status 1")
		cmd := exec.Command("curl", "-fL", "--connect-timeout", "20", "--max-time", "180",
			"-o", tmp, "-w", "http=%{http_code} size=%{size_download}\n", url)
		out, err := cmd.CombinedOutput()
		log.Write(out)
		if err != nil {
			log.WriteString(fmt.Sprintf("curl %s: %v\n", asset, err))
			// wget fallback
			cmd = exec.Command("wget", "-O", tmp, url)
			out2, err2 := cmd.CombinedOutput()
			log.Write(out2)
			if err2 != nil {
				return fmt.Errorf("%s: download failed url=%s curl=%v wget=%v", asset, url, err, err2)
			}
		}
		st, err := os.Stat(tmp)
		if err != nil {
			return fmt.Errorf("%s: missing file after download: %w", asset, err)
		}
		if st.Size() < 1000 {
			return fmt.Errorf("%s: download too small (%d bytes) — check GitHub reachability / asset name", asset, st.Size())
		}
		_ = os.Chmod(tmp, 0o755)
		if err := exec.Command("cp", "-f", tmp, dest).Run(); err != nil {
			return fmt.Errorf("%s: install to %s: %w", asset, dest, err)
		}
		log.WriteString("updated " + dest + " (" + fmt.Sprintf("%d", st.Size()) + " bytes)\n")
		return nil
	}
	// Secondary unit: ExecStart=/usr/local/bin/netductor secondary agent
	// OpenWrt netductor-agent binary must not be installed on secondary VPS.
	nodeAsset := "netductor-linux-" + arch
	if err := fetch(nodeAsset, "/usr/local/bin/netductor"); err != nil {
		log.WriteString(err.Error() + "\n")
		return false, log.String()
	}
	_ = os.WriteFile("/etc/netductor/VERSION", []byte(tag+"\n"), 0o644)
	_ = exec.Command("systemctl", "try-restart", "sing-box").Run()
	// Delay restart so reportCmdDone can reach primary first.
	go func() {
		time.Sleep(5 * time.Second)
		_ = exec.Command("systemctl", "restart", "netductor-secondary-agent").Run()
	}()
	// Day-2: re-assert secondary firewall + watchdog after binary replace.
	out, err := exec.Command("/usr/local/bin/netductor", "firewall", "apply", "secondary").CombinedOutput()
	if err != nil {
		log.WriteString(fmt.Sprintf("firewall apply secondary: %v %s\n", err, strings.TrimSpace(string(out))))
	} else {
		log.WriteString("firewall apply secondary ok\n")
	}
	out, err = exec.Command("/usr/local/bin/netductor", "stack", "watchdog-install").CombinedOutput()
	if err != nil {
		log.WriteString(fmt.Sprintf("watchdog-install: %v %s\n", err, strings.TrimSpace(string(out))))
	} else {
		log.WriteString("watchdog-install ok\n")
	}
	log.WriteString("DONE v" + tag + " (node binary; unit=netductor secondary agent)\n")
	return true, log.String()
}

func secondaryBackupLocal() (string, error) {
	// prefer CLI if present (same binary often)
	out, err := exec.Command("/usr/local/bin/netductor", "backup", "secondary-local").CombinedOutput()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("backup secondary-local: %w (%s)", err, strings.TrimSpace(string(out)))
}

func secondaryMTLSRefresh() (bool, string) {
	// load token + core URL from secrets
	tok := ""
	core := ""
	if b, err := os.ReadFile("/etc/netductor/secrets/secondary_agent_token"); err == nil {
		tok = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("/etc/netductor/secrets/secondary_core_url"); err == nil {
		core = strings.TrimSpace(string(b))
	}
	if tok == "" || core == "" {
		return false, "missing secondary_agent_token or secondary_core_url"
	}
	core = strings.TrimRight(core, "/")
	client := &http.Client{Timeout: 60 * time.Second}
	// First refresh may still need existing certs; if none, try without client cert only for material endpoint is unsafe —
	// primary requires mTLS. Surface clear error.
	tlsCfg, err := mtls.ClientTLSConfig()
	if err != nil {
		return false, "mtls client material missing — re-seed from primary deploy/secondary pack: " + err.Error()
	}
	client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	req, err := http.NewRequest(http.MethodGet, core+"/api/secondary/agent/mtls/material", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var mat struct {
		CA   string `json:"ca_pem"`
		Cert string `json:"cert_pem"`
		Key  string `json:"key_pem"`
	}
	if json.Unmarshal(body, &mat) != nil || mat.Cert == "" {
		return false, "bad material json"
	}
	dir := "/etc/netductor/secrets/mtls"
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(dir+"/ca.crt", []byte(mat.CA), 0o600)
	_ = os.WriteFile(dir+"/client.crt", []byte(mat.Cert), 0o600)
	_ = os.WriteFile(dir+"/client.key", []byte(mat.Key), 0o600)
	go func() {
		time.Sleep(2 * time.Second)
		_ = exec.Command("systemctl", "restart", "netductor-secondary-agent").Start()
	}()
	return true, "mtls material written; restarting agent"
}

func secondaryBackupPull() (bool, string) {
	tok := ""
	core := ""
	if b, err := os.ReadFile("/etc/netductor/secrets/secondary_agent_token"); err == nil {
		tok = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("/etc/netductor/secrets/secondary_core_url"); err == nil {
		core = strings.TrimSpace(string(b))
	}
	if tok == "" || core == "" {
		return false, "missing secondary_agent_token or secondary_core_url"
	}
	core = strings.TrimRight(core, "/")
	client := &http.Client{Timeout: 10 * time.Minute}
	tlsCfg, err := mtls.ClientTLSConfig()
	if err != nil {
		return false, "mtls client material missing/unreadable (need secrets/mtls ca+client cert+key): " + err.Error()
	}
	client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	req, err := http.NewRequest(http.MethodGet, core+"/api/secondary/agent/backup/latest", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	var resp *http.Response
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, lastErr = client.Do(req)
		if lastErr == nil {
			break
		}
		// Primary may restart API during stack apply — brief EOF/reset is common.
		time.Sleep(time.Duration(attempt*2) * time.Second)
		// rebuild request (Body was nil; URL reusable)
		req, _ = http.NewRequest(http.MethodGet, core+"/api/secondary/agent/backup/latest", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if lastErr != nil {
		return false, fmt.Sprintf("after retries: %v", lastErr)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	name := resp.Header.Get("X-Netductor-Backup-Name")
	if name == "" {
		name = "netductor-latest.ndenc"
	}
	dir := "/var/lib/netductor/backups/peers/core"
	_ = os.MkdirAll(dir, 0o700)
	outPath := filepath.Join(dir, name)
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return false, err.Error()
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return false, err.Error()
	}
	// key sidecar
	req2, _ := http.NewRequest(http.MethodGet, core+"/api/secondary/agent/backup/key", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	if resp2, err := client.Do(req2); err == nil {
		defer resp2.Body.Close()
		if resp2.StatusCode == 200 {
			kb, _ := io.ReadAll(resp2.Body)
			_ = os.WriteFile(filepath.Join(dir, "BACKUP_KEY.txt"), kb, 0o600)
		}
	}
	if comps := resp.Header.Get("X-Netductor-Components"); comps != "" {
		_ = os.WriteFile(filepath.Join(dir, "COMPONENTS.txt"), []byte(strings.ReplaceAll(comps, ",", "\n")+"\n"), 0o644)
	}
	prunePeerArchives(dir, 14)
	return true, "stored " + outPath
}

func prunePeerArchives(dir string, keep int) {
	if keep < 1 {
		keep = 1
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".ndenc") || strings.HasSuffix(n, ".tar.gz") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	if len(names) <= keep {
		return
	}
	for _, n := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(dir, n))
	}
}

func applyFailoverPolicy(raw json.RawMessage) error {
	dir := "/etc/netductor/svc-paths"
	_ = os.MkdirAll(dir, 0o755)
	path := dir + "/failover-policy.json"
	// pretty-print stable
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return os.WriteFile(path, raw, 0o644)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
