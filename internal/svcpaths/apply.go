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

// hostRole is shared by ProbeHealth and EnforceState.
func hostRole() string {
	role := "unknown"
	if hasIP(IfaceSP, AddrP_SP) || hasIP(IfacePS, AddrP_PS) {
		role = "primary"
	}
	if hasIP(IfaceSP, AddrS_SP) || hasIP(IfacePS, AddrS_PS) {
		role = "secondary"
	}
	for _, s := range SlotTable() {
		if hasIP(s.IfaceSP, s.AddrP_SP) || hasIP(s.IfacePS, s.AddrP_PS) {
			role = "primary"
		}
		if hasIP(s.IfaceSP, s.AddrS_SP) || hasIP(s.IfacePS, s.AddrS_PS) {
			role = "secondary"
		}
	}
	if role == "unknown" {
		if _, err := os.Stat(coreURLPath); err == nil {
			role = "secondary"
		} else if _, err := os.Stat(filepath.Join(KeysDir, "material.json")); err == nil {
			role = "primary"
		}
	}
	return role
}

// ProbeHealth writes HealthJSON (SP/PS/VLESS). Safe on primary and secondary.
func ProbeHealth() error {
	role := hostRole()
	sp, ps := 0, 0
	pSP, sSP, pPS, sPS := AddrP_SP, AddrS_SP, AddrP_PS, AddrS_PS
	if m, err := LoadMaterialStrict(); err == nil {
		a, b, c, d, _, _, _, _, _, _ := m.resolvedAddrs()
		pSP, sSP, pPS, sPS = a, b, c, d
	}
	if role == "primary" {
		if pingOK(sSP) {
			sp = 1
		}
		if pingOK(sPS) {
			ps = 1
		}
		if idx, err := LoadPeersIndex(); err == nil {
			for _, peer := range idx.Peers {
				if peer.State == "removed" {
					continue
				}
				if pingOK(peer.Slot.AddrS_SP) {
					sp = 1
				}
				if pingOK(peer.Slot.AddrS_PS) {
					ps = 1
				}
			}
		}
	} else {
		if pingOK(pSP) {
			sp = 1
		}
		if pingOK(pPS) {
			ps = 1
		}
	}
	vless := 0
	if role == "secondary" {
		// Local Reality/VLESS inbound on :443 — independent of SP/PS tunnel health.
		if probeTCP("127.0.0.1", "443", 2*time.Second) {
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
	if hostRole() != "secondary" {
		return nil
	}
	if err := applyAgentURL(s.AgentURLMode); err != nil {
		return err
	}
	return applyUserUplink(s.UserPath)
}

func applyAgentURL(mode string) error {
	_ = os.MkdirAll(filepath.Dir(coreURLPath), 0o700)
	// Tunnel-only: SP default, PS when SP path is down. Never public WAN :8789.
	pSP, pPS := AddrP_SP, AddrP_PS
	if m, err := LoadMaterialStrict(); err == nil {
		a, _, c, _, _, _, _, _, _, _ := m.resolvedAddrs()
		pSP, pPS = a, c
	}
	want := "https://" + pSP + ":8789"
	if mode == "public" || mode == "ps" || mode == "tunnel_ps" {
		want = "https://" + pPS + ":8789"
	}
	cur := strings.TrimSpace(string(mustRead(coreURLPath)))
	if cur == want {
		return nil
	}
	if err := os.WriteFile(coreURLPath, []byte(want+"\n"), 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "svc-paths apply: agent URL → %s (tunnel-only)\n", want)
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
	// Slot-aware: SP/PS primary addresses from this host's material (slot 1–3 ≠ slot 0 constants).
	pSP, pPS := PrimaryTunnelAddrs()
	target := pubHost
	switch userPath {
	case "sp":
		target = pSP
	case "ps":
		target = pPS
	case "vless":
		target = pubHost
	case "degraded":
		// Keep last known target; do not rewrite uplink to a dead path.
		return nil
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
	prev := mustRead(singBoxConfig)
	bak := singBoxConfig + ".bak"
	_ = os.WriteFile(bak, prev, 0o644)
	tmp := singBoxConfig + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, singBoxConfig); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "restart", "sing-box").CombinedOutput(); err != nil {
		_ = os.WriteFile(singBoxConfig, prev, 0o644)
		_ = exec.Command("systemctl", "restart", "sing-box").Run()
		return fmt.Errorf("restart sing-box after uplink apply: %w: %s (restored previous config)", err, strings.TrimSpace(string(out)))
	}
	if st, err := exec.Command("systemctl", "is-active", "sing-box").CombinedOutput(); err != nil || strings.TrimSpace(string(st)) != "active" {
		_ = os.WriteFile(singBoxConfig, prev, 0o644)
		_ = exec.Command("systemctl", "restart", "sing-box").Run()
		return fmt.Errorf("sing-box not active after uplink apply: %s (restored previous config)", strings.TrimSpace(string(st)))
	}
	_ = os.Remove(bak)
	return nil
}

// RunSecondaryCycle: probe → tick → apply. Serialized with flock (agent + timer share this).
func RunSecondaryCycle() (State, string, error) {
	unlock, err := acquireFailoverLock()
	if err != nil {
		return State{}, "", err
	}
	defer unlock()
	if err := ProbeHealth(); err != nil {
		return State{}, "", fmt.Errorf("probe health: %w", err)
	}
	s, sum, err := Tick()
	if err != nil {
		return s, sum, err
	}
	if err := EnforceState(s); err != nil {
		return s, sum, err
	}
	return s, sum, nil
}
