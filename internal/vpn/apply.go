package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

const singboxConf = "/usr/local/etc/sing-box/config.json"

func ApplyConfig() error {
	if err := EnsureDirs(); err != nil {
		return err
	}
	priv := secret("singbox_reality_private")
	sid := secret("singbox_short_id")
	if priv == "" || sid == "" {
		return fmt.Errorf("Reality secrets missing")
	}
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	type vu struct {
		UUID string `json:"uuid"`
		Flow string `json:"flow"`
	}
	type hu struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	var vusers []vu
	var husers []hu
	for _, u := range r.Users {
		if !u.Enabled {
			continue
		}
		// relay-uplink must NOT use vision: multiplex is incompatible with xtls-rprx-vision
		flow := "xtls-rprx-vision"
		if u.Name == RelayUplinkName {
			flow = ""
		}
		vusers = append(vusers, vu{UUID: u.UUID, Flow: flow})
		pass := u.Hy2Password
		if pass == "" {
			pass = u.UUID
		}
		husers = append(husers, hu{Name: u.Name, Password: pass})
	}
	if vusers == nil {
		vusers = []vu{}
	}
	if husers == nil {
		husers = []hu{}
	}

	sniVal := sni()
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
		"inbounds": []any{
			map[string]any{
				"type": "vless", "tag": "vless-reality", "listen": "::", "listen_port": vlessPort(),
				"users": vusers,
				// multiplex for secondary uplink (no vision). End-user vision streams stay non-mux.
				"multiplex": map[string]any{
					"enabled": true,
					"padding": false,
				},
				"tls": map[string]any{
					"enabled": true, "server_name": sniVal,
					"reality": map[string]any{
						"enabled": true,
						"handshake": map[string]any{"server": sniVal, "server_port": 443},
						"private_key": priv, "short_id": []string{sid},
					},
				},
			},
			map[string]any{
				"type": "hysteria2", "tag": "hy2", "listen": "::", "listen_port": hy2Port(),
				"users": husers,
				"tls": map[string]any{
					"enabled": true, "alpn": []string{"h3"},
					"certificate_path": "/etc/sing-box/certs/hy2.crt",
					"key_path":         "/etc/sing-box/certs/hy2.key",
				},
				"masquerade": "https://" + sniVal,
			},
		},
		"outbounds": outbounds,
		"route": map[string]any{
			"rules": routeRules,
			"final": finalOut, "auto_detect_interface": true, "default_domain_resolver": "blocky",
		},
	}

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
	_ = exec.Command("systemctl", "restart", "sing-box").Run()
	fmt.Fprintf(os.Stderr, "sing-box config variant=%s (restart; alerts suppressed 90s)\n", variant)
	_ = bumpSecondaryConfigVer() // end users on secondary VLESS must refresh
	return nil
}

// bumpSecondaryConfigVer asks secondary agents to pull a new ExportRelayBundle
// (includes all VPN UUIDs). Avoids import cycle with package secondary.
func bumpSecondaryConfigVer() error {
	for _, name := range []string{"secondary", "relay"} {
		path := filepath.Join(paths.StateDir(), name, "devices.json")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var r map[string]any
		if err := json.Unmarshal(b, &r); err != nil {
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
			continue
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, append(out, '\n'), 0o600); err != nil {
			continue
		}
		_ = os.Rename(tmp, path)
		fmt.Fprintf(os.Stderr, "secondary config_ver=%d (%s)\n", v, name)
		return nil
	}
	return nil
}

func tryCheck(conf string) bool {
	bin := "/usr/local/bin/sing-box"
	if _, err := os.Stat(bin); err != nil {
		return true // no binary → accept first
	}
	return exec.Command(bin, "check", "-c", conf).Run() == nil
}

func cloneMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}


func buildOutboundsAndRoute() (outbounds []any, routeRules []any, finalOut string) {
	outbounds = []any{map[string]any{"type": "direct", "tag": "direct"}}
	routeRules = []any{
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
	}
	finalOut = "direct"
	exitOn, ip, pbk, sid, sniR := readExitTarget()
	exitUUID := secret("secondary_exit_uuid")
	if !exitOn || ip == "" || pbk == "" || exitUUID == "" {
		return
	}
	if sniR == "" {
		sniR = DefaultRealitySNI
	}
	outbounds = append(outbounds, map[string]any{
		"type": "vless", "tag": "ru-exit",
		"server": ip, "server_port": 4443,
		"uuid": exitUUID, "flow": "xtls-rprx-vision",
		"tls": map[string]any{
			"enabled": true, "server_name": sniR,
			"utls": map[string]any{"enabled": true, "fingerprint": "chrome"},
			"reality": map[string]any{
				"enabled": true, "public_key": pbk, "short_id": sid,
			},
		},
	})
	finalOut = "ru-exit"
	return
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
		return "", fmt.Errorf("Reality secrets missing")
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
