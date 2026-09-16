package probes

import (
	"encoding/json"
	"os"
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
			map[string]any{"name": "api-health", "type": "http", "url": "https://127.0.0.1:8787/health", "insecure": true, "timeout": 3},
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
	return m
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
