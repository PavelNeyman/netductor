package svcpaths

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	singBoxConfig = "/usr/local/etc/sing-box/config.json"
	coreURLPath   = "/etc/netductor/secrets/secondary_core_url"
	coreURLPublic = "/etc/netductor/secrets/secondary_core_url.public"
)

// ProbeHealth writes HealthJSON (SP/PS/VLESS). Safe on primary and secondary.
func ProbeHealth() error {
	role := "unknown"
	if hasIP(IfaceSP, AddrP_SP) || hasIP(IfacePS, AddrP_PS) {
		role = "primary"
	}
	if hasIP(IfaceSP, AddrS_SP) || hasIP(IfacePS, AddrS_PS) {
		role = "secondary"
	}
	sp, ps := 0, 0
	if role == "primary" {
		if pingOK(AddrS_SP) {
			sp = 1
		}
		if pingOK(AddrS_PS) {
			ps = 1
		}
	} else {
		if pingOK(AddrP_SP) {
			sp = 1
		}
		if pingOK(AddrP_PS) {
			ps = 1
		}
	}
	vless := 0
	if role == "secondary" {
		if probeTCP(publicPrimaryHost(), "443", 3*time.Second) {
			vless = 1
		}
	}
	_ = os.MkdirAll(filepath.Dir(HealthJSON), 0o755)
	body := fmt.Sprintf(`{"ts":"%s","role":"%s","svc_sp_up":%d,"svc_ps_up":%d,"vless_tcp443":%d}`+"\n",
		time.Now().UTC().Format(time.RFC3339), role, sp, ps, vless)
	return os.WriteFile(HealthJSON, []byte(body), 0o644)
}

func pingOK(ip string) bool {
	return exec.Command("ping", "-c", "1", "-W", "2", ip).Run() == nil
}

func probeTCP(host, port string, d time.Duration) bool {
	if host == "" {
		return false
	}
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), d)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func publicPrimaryHost() string {
	for _, p := range []string{coreURLPublic, coreURLPath} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(b))
		s = strings.TrimPrefix(s, "https://")
		s = strings.TrimPrefix(s, "http://")
		if i := strings.IndexAny(s, ":/"); i > 0 {
			s = s[:i]
		}
		// skip tunnel address
		if s != "" && s != AddrP_SP && s != AddrS_SP {
			return s
		}
	}
	return ""
}

// Apply enforces desired user_path and agent_url_mode on this host (secondary).
func EnforceState(s State) error {
	role := "unknown"
	if hasIP(IfaceSP, AddrS_SP) || hasIP(IfacePS, AddrS_PS) {
		role = "secondary"
	}
	if role != "secondary" {
		return nil
	}
	if err := applyAgentURL(s.AgentURLMode); err != nil {
		return err
	}
	return applyUserUplink(s.UserPath)
}

func applyAgentURL(mode string) error {
	_ = os.MkdirAll(filepath.Dir(coreURLPath), 0o700)
	pub := strings.TrimSpace(string(mustRead(coreURLPublic)))
	if pub == "" {
		if h := publicPrimaryHost(); h != "" {
			pub = "https://" + h + ":8789"
		}
	}
	tunnel := "https://" + AddrP_SP + ":8789"
	want := tunnel
	if mode == "public" && pub != "" {
		want = pub
	}
	cur := strings.TrimSpace(string(mustRead(coreURLPath)))
	if cur == want {
		return nil
	}
	if err := os.WriteFile(coreURLPath, []byte(want+"\n"), 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "svc-paths apply: agent URL → %s\n", want)
	return nil
}

func mustRead(p string) []byte {
	b, _ := os.ReadFile(p)
	return b
}

func applyUserUplink(userPath string) error {
	b, err := os.ReadFile(singBoxConfig)
	if err != nil {
		return nil // sing-box not installed
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		return err
	}
	outs, _ := cfg["outbounds"].([]any)
	if outs == nil {
		return nil
	}
	pubHost := publicPrimaryHost()
	target := pubHost
	if userPath == "sp" || userPath == "ps" {
		target = AddrP_SP // user bulk via SP to primary Reality
	}
	if target == "" {
		return nil
	}
	changed := false
	for _, o := range outs {
		m, ok := o.(map[string]any)
		if !ok {
			continue
		}
		if m["tag"] != "uplink" {
			continue
		}
		cur, _ := m["server"].(string)
		if cur != target {
			m["server"] = target
			changed = true
			fmt.Fprintf(os.Stderr, "svc-paths apply: uplink server %s → %s (user_path=%s)\n", cur, target, userPath)
		}
	}
	if !changed {
		return nil
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(singBoxConfig, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "restart", "sing-box").Run()
	return nil
}

// RunSecondaryCycle: probe → tick → apply. Intended on secondary (agent + timer).
func RunSecondaryCycle() (State, string, error) {
	_ = ProbeHealth()
	s, sum, err := Tick()
	if err != nil {
		return s, sum, err
	}
	if err := EnforceState(s); err != nil {
		return s, sum, err
	}
	return s, sum, nil
}
