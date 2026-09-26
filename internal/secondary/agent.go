package secondary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/paths"
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
	coreBase = strings.TrimRight(coreBase, "/")
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
	upOK := probePrimaryUplink(coreBase)
	if upOK {
		uplinkFailStreak = 0
	} else {
		uplinkFailStreak++
		fmt.Fprintf(os.Stderr, "uplink probe primary:443 fail streak=%d\n", uplinkFailStreak)
		// after 3 consecutive fails (~1.5min at 30s tick): restart sing-box to clear stuck mux
		if uplinkFailStreak >= 3 && sbOK {
			fmt.Fprintf(os.Stderr, "uplink watchdog: restarting sing-box\n")
			_ = exec.Command("systemctl", "restart", "sing-box").Run()
			uplinkFailStreak = 0
		}
	}
	cpu, memU, memT, load1 := sampleMetrics()
	mm := vpn.CollectMismatch(30)
	body, _ := json.Marshal(HeartbeatIn{
		PublicIP: ip, PBK: pub, SID: sid, SNI: sni,
		Version: "agent-1", SingBoxOK: sbOK, UplinkOK: upOK, ConfigVer: *applied,
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
		ConfigVer       int      `json:"config_ver"`
		NeedSync        bool     `json:"need_sync"`
		DesiredHostname string   `json:"desired_hostname"`
		Commands        []string `json:"commands"`
	}
	_ = json.Unmarshal(raw, &hr)
	if hn := strings.TrimSpace(hr.DesiredHostname); hn != "" {
		_ = applyHostname(hn)
	}
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
		Version: "agent-1", SingBoxOK: sbOK, UplinkOK: upOK, ConfigVer: applied,
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
		return secondaryUpgrade()
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


// secondaryUpgrade: apt + binary replace without shell. Agent restart deferred in-process.
func secondaryUpgrade() (bool, string) {
	var log strings.Builder
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		out, err := cmd.CombinedOutput()
		log.Write(out)
		if err != nil {
			log.WriteString(err.Error())
			log.WriteByte('\n')
		}
	}
	run("apt-get", "update", "-qq")
	run("apt-get", "-y", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", "upgrade")
	ver := VersionHint()
	url := "https://github.com/PavelNeyman/netductor/releases/download/v" + ver + "/netductor-linux-amd64"
	tmp := "/tmp/nd-upgrade.bin"
	run("wget", "-qO", tmp, url)
	if st, err := os.Stat(tmp); err == nil && st.Size() > 1000 {
		_ = os.Chmod(tmp, 0o755)
		_ = exec.Command("cp", tmp, "/usr/local/bin/netductor").Run()
	}
	_ = exec.Command("systemctl", "restart", "sing-box").Run()
	go func() {
		time.Sleep(45 * time.Second)
		_ = exec.Command("systemctl", "restart", "netductor-secondary-agent").Run()
	}()
	log.WriteString("DONE\n")
	return true, log.String()
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
	if tlsCfg, err := mtls.ClientTLSConfig(); err == nil {
		client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	}
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
	if tlsCfg, err := mtls.ClientTLSConfig(); err == nil {
		client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	}
	req, err := http.NewRequest(http.MethodGet, core+"/api/secondary/agent/backup/latest", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
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
	return true, "stored " + outPath
}
