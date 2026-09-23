package probes

import (
	"encoding/json"
	"os"
	"strings"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func CfgPath() string { return paths.ProbesCfg() }

func Default() map[string]any {
	return map[string]any{
		"probes": []any{
			map[string]any{"name": "dns-blocky", "type": "tcp", "host": "127.0.0.1", "port": 53, "timeout": 2},
			map[string]any{"name": "vless", "type": "tcp", "host": "127.0.0.1", "port": 443, "timeout": 2},
			map[string]any{"name": "hy2", "type": "udp", "host": "127.0.0.1", "port": 8443, "timeout": 2},
			map[string]any{"name": "api-health", "type": "http", "url": "http://127.0.0.1:8787/health", "timeout": 3},
		},
		"alerts": map[string]any{
			"cpu_pct": 90, "mem_pct": 92, "disk_pct": 90,
			"service_not_active": true, "probe_fail": true,
			"service_down": true, "relay_offline": true,
			"cooldown_sec": 900,
		},
	}
}

func Load() map[string]any {
	b, err := os.ReadFile(CfgPath())
	if err != nil {
		return Default()
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return Default()
	}
	// migrate: API is HTTP-only on loopback; old defaults used https://
	if fixAPIHealthHTTPS(m) {
		_ = Save(m)
	}
	return m
}

func fixAPIHealthHTTPS(m map[string]any) bool {
	probes, ok := m["probes"].([]any)
	if !ok {
		return false
	}
	changed := false
	for i, p := range probes {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		name, _ := pm["name"].(string)
		url, _ := pm["url"].(string)
		if name == "api-health" && strings.HasPrefix(url, "https://127.0.0.1:8787") {
			pm["url"] = "http://127.0.0.1:8787/health"
			delete(pm, "insecure")
			probes[i] = pm
			changed = true
		}
	}
	if changed {
		m["probes"] = probes
	}
	return changed
}

func Save(cfg map[string]any) error {
	path := CfgPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
