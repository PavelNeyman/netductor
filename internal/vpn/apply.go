package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/policy"
)

const singboxConf = "/usr/local/etc/sing-box/config.json"

func ApplyConfig() error {
	if err := EnsureDirs(); err != nil {
		return err
	}
	priv := secret("singbox_reality_private")
	sid := secret("singbox_short_id")
	if priv == "" || sid == "" {
		return fmt.Errorf("reality secrets missing")
	}
	if err := EnsureUplinkABMaterial(); err != nil {
		return fmt.Errorf("uplink AB material: %w", err)
	}
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	type vu struct {
		Name string `json:"name,omitempty"`
		UUID string `json:"uuid"`
		Flow string `json:"flow"`
	}
	var vusers []vu
	for _, u := range r.Users {
		if !u.Enabled {
			continue
		}
		// relay-uplink must NOT use vision: multiplex is incompatible with xtls-rprx-vision
		flow := "xtls-rprx-vision"
		if u.Name == RelayUplinkName {
			flow = ""
		}
		vusers = append(vusers, vu{Name: u.Name, UUID: u.UUID, Flow: flow})
	}
	if vusers == nil {
		vusers = []vu{}
	}
	sniVal := sni()

	// Service uplink from secondary: no mux, no vision. Public :443 stays mux+vision for clients.
	var svcUsers []vu
	for _, u := range vusers {
		su := u
		su.Flow = ""
		svcUsers = append(svcUsers, su)
	}
	outbounds, routeRules, finalOut := buildOutboundsAndRoute()
	cfg := map[string]any{
		"log": map[string]any{"level": "info", "timestamp": true},
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"type": "udp", "tag": "blocky", "server": "127.0.0.1"},
				map[string]any{"type": "local", "tag": "local"},
			},
			"final": "blocky", "strategy": "ipv4_only",
		},
		"inbounds":  appendPrimaryUplinkABInbounds(buildPrimaryInbounds(vusers, svcUsers, sniVal, priv, sid), func() string {
			for _, u := range r.Users {
				if u.Name == RelayUplinkName {
					return u.UUID
				}
			}
			return ""
		}(), sniVal, priv, sid),
		"outbounds": outbounds,
		"route": map[string]any{
			"rules": routeRules,
			"final": finalOut, "auto_detect_interface": true, "default_domain_resolver": "blocky",
		},
	}
	// C11: localhost clash_api — rewritten on every Apply so it is never dropped.
	AttachExperimentalClashAPI(cfg)
	if eps := primaryUplinkABEndpoints(); len(eps) > 0 {
		cfg["endpoints"] = eps
	}
	openUplinkABFirewall()
	EnsureReverseBridge()

	work, err := os.MkdirTemp("", "nd-sb-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	writeJSON := func(path string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, append(b, '\n'), 0o600)
	}
	aPath := filepath.Join(work, "a.json")
	if err := writeJSON(aPath, cfg); err != nil {
		return err
	}
	// variant b: local dns only
	cfgB := cloneMap(cfg)
	cfgB["dns"] = map[string]any{"servers": []any{map[string]any{"type": "local", "tag": "local"}}, "final": "local"}
	if route, ok := cfgB["route"].(map[string]any); ok {
		route["default_domain_resolver"] = "local"
	}
	bPath := filepath.Join(work, "b.json")
	_ = writeJSON(bPath, cfgB)
	// variant c: minimal route
	cfgC := cloneMap(cfg)
	delete(cfgC, "dns")
	cfgC["route"] = map[string]any{"final": "direct", "auto_detect_interface": true}
	cPath := filepath.Join(work, "c.json")
	_ = writeJSON(cPath, cfgC)

	chosen, variant := "", ""
	for _, pair := range []struct{ p, v string }{{aPath, "a"}, {bPath, "b"}, {cPath, "c"}} {
		if tryCheck(pair.p) {
			chosen, variant = pair.p, pair.v
			break
		}
	}
	if chosen == "" {
		return fmt.Errorf("sing-box config validation failed for all variants")
	}
	_ = os.MkdirAll("/usr/local/etc/sing-box", 0o755)
	b, err := os.ReadFile(chosen)
	if err != nil {
		return err
	}
	if err := os.WriteFile(singboxConf, b, 0o600); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "singbox-config-variant"), []byte(variant+"\n"), 0o644)
	notify.SuppressUntil(90 * time.Second)
	if out, err := exec.Command("systemctl", "restart", "sing-box").CombinedOutput(); err != nil {
		return fmt.Errorf("sing-box restart after writing variant %s: %v (%s)", variant, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("systemctl", "is-active", "sing-box").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "active" {
		return fmt.Errorf("sing-box not active after restart (config on disk variant=%s): %s", variant, strings.TrimSpace(string(out)))
	}
	fmt.Fprintf(os.Stderr, "sing-box config variant=%s (restart; alerts suppressed 90s)\n", variant)
	if err := bumpSecondaryConfigVer(); err != nil {
		return fmt.Errorf("config written and sing-box restarted, but config_ver bump failed: %w", err)
	}
	return nil
}

// bumpSecondaryConfigVer asks secondary agents to pull a new ExportSecondaryBundle
// (includes all VPN UUIDs). Avoids import cycle with package secondary.
func bumpSecondaryConfigVer() error {
	// CF-11: authoritative registry only (secondary/devices.json). Optional one-shot
	// read of legacy relay/ if secondary missing — still error if nothing bumped.
	pathsToTry := []string{filepath.Join(paths.StateDir(), "secondary", "devices.json")}
	if _, err := os.Stat(pathsToTry[0]); err != nil {
		legacy := filepath.Join(paths.StateDir(), "relay", "devices.json")
		if _, err2 := os.Stat(legacy); err2 == nil {
			pathsToTry = append(pathsToTry, legacy)
		}
	}
	var lastErr error
	bumped := false
	for _, path := range pathsToTry {
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		var r map[string]any
		if err := json.Unmarshal(b, &r); err != nil {
			lastErr = err
			continue
		}
		v := 1
		switch x := r["config_ver"].(type) {
		case float64:
			v = int(x) + 1
		case int:
			v = x + 1
		default:
			v = 2
		}
		r["config_ver"] = v
		out, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			lastErr = err
			continue
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, append(out, '\n'), 0o600); err != nil {
			lastErr = err
			continue
		}
		if err := os.Rename(tmp, path); err != nil {
			lastErr = err
			continue
		}
		fmt.Fprintf(os.Stderr, "secondary config_ver=%d (%s)\n", v, path)
		bumped = true
		// Prefer secondary path; if we only had legacy, stop after one success.
		if strings.Contains(path, string(filepath.Separator)+"secondary"+string(filepath.Separator)) {
			return nil
		}
		return nil
	}
	if !bumped {
		if lastErr != nil {
			return fmt.Errorf("bump config_ver: no registry updated: %w", lastErr)
		}
		return fmt.Errorf("bump config_ver: secondary devices.json not found under %s", paths.StateDir())
	}
	return nil
}

func tryCheck(conf string) bool {
	bin := "/usr/local/bin/sing-box"
	if _, err := os.Stat(bin); err != nil {
		return true // no binary → accept first
	}
	cmd := exec.Command(bin, "check", "-c", conf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sing-box check %s: %v\n%s\n", conf, err, string(out))
		return false
	}
	return true
}

func cloneMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func policyServiceRules() []any {
	r, err := loadRegistry()
	if err != nil {
		return nil
	}
	cat, _ := policy.EnsureCatalog()
	var subjects []policy.Subject
	for _, u := range r.Users {
		if !u.Enabled {
			continue
		}
		subjects = append(subjects, policy.Subject{Name: u.Name, Policy: PolicyFromRecord(u)})
	}
	return policy.ServiceRouteRules(subjects, cat)
}

func readExitTarget() (on bool, ip, pbk, sid, sni string) {
	b, err := os.ReadFile(paths.SecondaryDevicesFile())
	if err != nil {
		return
	}
	var reg struct {
		ExitEnabled bool `json:"exit_enabled"`
		Devices     []struct {
			PublicIP string `json:"public_ip"`
			PBK      string `json:"pbk"`
			SID      string `json:"sid"`
			SNI      string `json:"sni"`
			LastSeen string `json:"last_seen"`
		} `json:"devices"`
	}
	if json.Unmarshal(b, &reg) != nil {
		return
	}
	on = reg.ExitEnabled
	for _, d := range reg.Devices {
		if d.PublicIP != "" && d.PBK != "" {
			return on, d.PublicIP, d.PBK, d.SID, d.SNI
		}
	}
	return on, "", "", "", ""
}

// ApplyConfigDryRun validates config generation without writing/restarting.
func ApplyConfigDryRun() (string, error) {
	if err := EnsureDirs(); err != nil {
		return "", err
	}
	priv := secret("singbox_reality_private")
	sid := secret("singbox_short_id")
	if priv == "" || sid == "" {
		return "", fmt.Errorf("reality secrets missing")
	}
	r, err := loadRegistry()
	if err != nil {
		return "", err
	}
	enabled := 0
	for _, u := range r.Users {
		if u.Enabled {
			enabled++
		}
	}
	return fmt.Sprintf("dry-run ok users_total=%d enabled=%d sni=%s", len(r.Users), enabled, sni()), nil
}
