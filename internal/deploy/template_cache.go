package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
)

// TemplateCacheDir is ~/.cache/netductor/agents/templates (next to agent binaries).
func TemplateCacheDir(agentDir string) string {
	if agentDir == "" {
		home, _ := os.UserHomeDir()
		agentDir = filepath.Join(home, ".cache", "netductor", "agents")
	}
	return filepath.Join(agentDir, "templates")
}

// CacheDefaultTemplates writes built-in default template JSON for offline deploy.
func CacheDefaultTemplates(agentDir string) (string, error) {
	dir := TemplateCacheDir(agentDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	edge.EnsureDefaultTemplate()
	// Prefer reading from edge package disk if primary-local; else synthesize same shape.
	t, err := edge.GetTemplate("default")
	if err != nil {
		t = edge.Template{
			"id":   "default",
			"role": "site",
			"guest": map[string]any{"enabled": false, "ssid": "Guest", "hidden": false},
			"network": map[string]any{
				"lan_ip": "192.168.50.1", "lan_mask": "255.255.255.0", "dhcp": true,
			},
			"wifi": map[string]any{"ssid": "Netductor", "encryption": "psk2", "key": ""},
			"vpn": map[string]any{
				"enabled": true, "mode": "tun", "fallback": "wan", "soft_fallback": true, "dns": "vpn",
			},
			"updated": time.Now().Unix(),
		}
	}
	path := filepath.Join(dir, "default.json")
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// PullTemplatesFromPrimary copies edge template JSON files from primary via SSH.
func PullTemplatesFromPrimary(primaryHost, primaryUser, primaryKey, keyPass, agentDir string) (int, error) {
	if primaryHost == "" || primaryKey == "" {
		return 0, fmt.Errorf("primary host and key required")
	}
	if primaryUser == "" {
		primaryUser = "root"
	}
	dir := TemplateCacheDir(agentDir)
	_ = os.MkdirAll(dir, 0o755)
	// List template ids on primary, then fetch each (paths under /var/lib/netductor/edge/templates).
	script := `ls /var/lib/netductor/edge/templates/*.json 2>/dev/null | while read f; do echo FILE:$(basename "$f"); base64 -w0 "$f" 2>/dev/null || base64 "$f" | tr -d '\n'; echo; done`
	out, err := runSSHOnPort(day2SSHPort(), "", primaryKey, primaryUser, primaryHost, script, keyPass)
	if err != nil {
		return 0, fmt.Errorf("list templates: %w\n%s", err, out)
	}
	n := 0
	lines := strings.Split(out, "\n")
	var curName string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Warning:") {
			continue
		}
		if strings.HasPrefix(line, "FILE:") {
			curName = strings.TrimPrefix(line, "FILE:")
			continue
		}
		if curName == "" {
			continue
		}
		raw, err := decodeB64(line)
		if err != nil || len(raw) == 0 {
			curName = ""
			continue
		}
		path := filepath.Join(dir, curName)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			curName = ""
			continue
		}
		n++
		curName = ""
	}
	if n == 0 {
		// Fallback: local default
		if _, err := CacheDefaultTemplates(agentDir); err == nil {
			return 1, nil
		}
	}
	return n, nil
}

// LoadCachedTemplate reads a template from the agent-adjacent cache.
func LoadCachedTemplate(agentDir, id string) (edge.Template, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "default"
	}
	path := filepath.Join(TemplateCacheDir(agentDir), id+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t edge.Template
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return t, nil
}
