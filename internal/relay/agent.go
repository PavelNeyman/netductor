package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// AgentLoop runs on RU VPS: heartbeat + pull config when version drifts.
func AgentLoop(coreBase, token string, interval time.Duration) {
	if interval < 10*time.Second {
		interval = 30 * time.Second
	}
	client := &http.Client{Timeout: 20 * time.Second}
	applied := 0
	var lastDone string
	var lastOK bool
	var lastLog string
	for {
		_ = agentTick(client, coreBase, token, &applied, &lastDone, &lastOK, &lastLog)
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
	cpu, memU, memT, load1 := sampleMetrics()
	mm := vpn.CollectMismatch(30)
	body, _ := json.Marshal(HeartbeatIn{
		PublicIP: ip, PBK: pub, SID: sid, SNI: sni,
		Version: "agent-1", SingBoxOK: sbOK, ConfigVer: *applied,
		CPUPercent: cpu, MemUsedMB: memU, MemTotalMB: memT, Load1: load1,
		CmdDone: *lastDone, CmdOK: *lastOK, CmdLog: *lastLog,
		MismatchTotal: mm.Total, MismatchByIP: mm.ByIP,
	})
	*lastDone, *lastOK, *lastLog = "", false, ""
	req, err := http.NewRequest(http.MethodPost, coreBase+"/api/relay/agent/heartbeat", bytes.NewReader(body))
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
	req, err := http.NewRequest(http.MethodGet, coreBase+"/api/relay/agent/config", nil)
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
	var b vpn.RelayBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return err
	}
	priv := readSecret("singbox_reality_private")
	sid := readSecret("singbox_short_id")
	if err := vpn.WriteRelaySingBox(&b, priv, sid); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "restart", "sing-box").Run()
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "relay", "bundle.json"), raw, 0o600)
	return nil
}

func readSecret(name string) string {
	b, _ := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", name))
	return strings.TrimSpace(string(b))
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
	body, _ := json.Marshal(HeartbeatIn{
		PublicIP: ip, PBK: pub, SID: sid, SNI: sni,
		Version: "agent-1", SingBoxOK: sbOK, ConfigVer: applied,
		CPUPercent: cpu, MemUsedMB: memU, MemTotalMB: memT, Load1: load1,
		CmdDone: cmd, CmdOK: ok, CmdLog: log,
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(coreBase, "/")+"/api/relay/agent/heartbeat", bytes.NewReader(body))
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

	switch strings.TrimSpace(cmd) {
	case "reboot":
		go func() {
			time.Sleep(2 * time.Second)
			_ = exec.Command("systemctl", "reboot").Run()
		}()
		return true, "reboot scheduled in 2s"
	case "upgrade":
		out, err := exec.Command("bash", "-c", `export DEBIAN_FRONTEND=noninteractive
apt-get update -qq 2>&1 | tail -5
apt-get -y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold upgrade 2>&1 | tail -30
wget -qO /tmp/nd.bin https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64 && cp /tmp/nd.bin /usr/local/bin/netductor
systemctl restart sing-box 2>&1 || true
# restart agent later so this process can report cmd_done on next heartbeat first
nohup bash -c 'sleep 45; systemctl restart netductor-relay-agent' >/dev/null 2>&1 &
echo DONE
`).CombinedOutput()
		return err == nil, string(out)
	case "metrics":
		return true, "metrics on next heartbeat"
	case "journal":
		out, err := exec.Command("journalctl", "-u", "sing-box", "-u", "netductor-relay-agent", "-n", "60", "--no-pager", "-o", "short-iso").CombinedOutput()
		return err == nil || len(out) > 0, string(out)
	default:
		if strings.HasPrefix(cmd, "restart:") {
			unit := strings.TrimPrefix(cmd, "restart:")
			out, err := exec.Command("systemctl", "restart", unit).CombinedOutput()
			return err == nil, string(out)
		}

		return false, "unknown cmd: " + cmd
	}
}

