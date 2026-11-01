package svcpaths

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	m, errM := LoadMaterialForHost()
	if errM != nil {
		if !os.IsNotExist(errM) {
			return probeHealthInvalid("material load: " + errM.Error())
		}
		// no file: legacy slot 0
	} else if a, b, c, d, _, _, _, _, _, _, errR := resolveBootstrapParams(m); errR != nil {
		return probeHealthInvalid("material identity: " + errR.Error())
	} else {
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
	if err := os.MkdirAll(filepath.Dir(coreURLPath), 0o700); err != nil {
		return fmt.Errorf("create agent URL directory: %w", err)
	}
	// Tunnel-only: SP default, PS when SP path is down. Never public WAN :8789.
	pSP, pPS, addrErr := PrimaryTunnelAddrsStrict()
	if addrErr != nil {
		return fmt.Errorf("agent URL material: %w", addrErr)
	}
	want := "https://" + pSP + ":8789"
	if mode == "public" || mode == "ps" || mode == "tunnel_ps" {
		want = "https://" + pPS + ":8789"
	}
	curBytes, readErr := os.ReadFile(coreURLPath)
	if readErr == nil && strings.TrimSpace(string(curBytes)) == want {
		return nil
	}
	modeBits, uid, gid := os.FileMode(0o600), os.Getuid(), os.Getgid()
	info, statErr := os.Lstat(coreURLPath)
	if statErr == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("agent URL path is not a regular file")
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("agent URL ownership metadata unavailable")
		}
		modeBits, uid, gid = info.Mode().Perm(), int(st.Uid), int(st.Gid)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect agent URL file: %w", statErr)
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read agent URL file: %w", readErr)
	}
	if err := writeFileAtomic(coreURLPath, []byte(want+"\n"), modeBits, uid, gid); err != nil {
		return fmt.Errorf("atomically write agent URL: %w", err)
	}
	fmt.Fprintf(os.Stderr, "svc-paths apply: agent URL → %s (tunnel-only)\n", want)
	return nil
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
	// Slot-aware critical path: corrupt material must not silently target slot 0.
	pSP, pPS, addrErr := PrimaryTunnelAddrsStrict()
	if addrErr != nil {
		return fmt.Errorf("user uplink material: %w", addrErr)
	}
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
	info, err := os.Lstat(singBoxConfig)
	if err != nil {
		return fmt.Errorf("stat sing-box config before replacement: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("sing-box config path is not a regular file")
	}
	mode := info.Mode().Perm()
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("sing-box config ownership metadata unavailable")
	}
	uid, gid := int(stat.Uid), int(stat.Gid)
	prev := append([]byte(nil), b...)
	if err := writeFileAtomic(singBoxConfig, append(raw, '\n'), mode, uid, gid); err != nil {
		if restoreErr := writeFileAtomic(singBoxConfig, prev, mode, uid, gid); restoreErr != nil {
			return fmt.Errorf("atomically replace sing-box config: %w; restoring previous config also failed: %v", err, restoreErr)
		}
		return fmt.Errorf("atomically replace sing-box config: %w; previous config restored", err)
	}
	rollback := func(cause error) error {
		if restoreErr := writeFileAtomic(singBoxConfig, prev, mode, uid, gid); restoreErr != nil {
			return fmt.Errorf("%w; restore previous sing-box config failed: %v; manual recovery required", cause, restoreErr)
		}
		out, restartErr := exec.Command("systemctl", "restart", "sing-box").CombinedOutput()
		if restartErr != nil {
			return fmt.Errorf("%w; previous config restored but sing-box restart failed: %v: %s", cause, restartErr, strings.TrimSpace(string(out)))
		}
		st, activeErr := exec.Command("systemctl", "is-active", "sing-box").CombinedOutput()
		if activeErr != nil || strings.TrimSpace(string(st)) != "active" {
			return fmt.Errorf("%w; previous config restored but sing-box is not active: %s (%v)", cause, strings.TrimSpace(string(st)), activeErr)
		}
		return cause
	}
	out, restartErr := exec.Command("systemctl", "restart", "sing-box").CombinedOutput()
	if restartErr != nil {
		return rollback(fmt.Errorf("restart sing-box after uplink apply: %w: %s", restartErr, strings.TrimSpace(string(out))))
	}
	st, activeErr := exec.Command("systemctl", "is-active", "sing-box").CombinedOutput()
	if activeErr != nil || strings.TrimSpace(string(st)) != "active" {
		return rollback(fmt.Errorf("sing-box not active after uplink apply: %s (%v)", strings.TrimSpace(string(st)), activeErr))
	}
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

func probeHealthInvalid(why string) error {
	_ = os.MkdirAll(filepath.Dir(HealthJSON), 0o755)
	body := fmt.Sprintf(`{"ts":"%s","role":"unknown","svc_sp_up":0,"svc_ps_up":0,"vless_tcp443":0,"note":"material-invalid","error":%q}`+"\n",
		time.Now().UTC().Format(time.RFC3339), why)
	_ = os.WriteFile(HealthJSON, []byte(body), 0o644)
	return fmt.Errorf("probe health: %s", why)
}


// writeFileAtomic replaces a file without exposing a partially written config.
// The temporary file is created in the destination directory so rename is atomic.
func writeFileAtomic(path string, data []byte, mode os.FileMode, uid, gid int) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".netductor-atomic-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		_ = os.Remove(tmp)
	}()
	if err := f.Chown(uid, gid); err != nil {
		_ = f.Close()
		return fmt.Errorf("preserve file ownership: %w", err)
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open parent directory for sync: %w", err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("sync parent directory: %w", err)
	}
	return nil
}
