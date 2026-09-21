package edge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func tmplRoot() string {
	_ = ensure()
	d := filepath.Join(filepath.Dir(deviceDir("_")), "templates")
	_ = os.MkdirAll(d, 0o755)
	return d
}

type Template map[string]any

func sanitizeID(id string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, id)
}

func SaveTemplate(id string, t Template) error {
	id = sanitizeID(id)
	if id == "" {
		return fmt.Errorf("empty id")
	}
	t["id"] = id
	t["updated"] = time.Now().Unix()
	b, _ := json.MarshalIndent(t, "", "  ")
	path := filepath.Join(tmplRoot(), id+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func GetTemplate(id string) (Template, error) {
	id = sanitizeID(id)
	b, err := os.ReadFile(filepath.Join(tmplRoot(), id+".json"))
	if err != nil {
		return nil, err
	}
	var t Template
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return t, nil
}

func ListTemplates() []Template {
	entries, err := os.ReadDir(tmplRoot())
	if err != nil {
		return nil
	}
	var out []Template
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		if t, err := GetTemplate(id); err == nil {
			out = append(out, t)
		}
	}
	return out
}

func DeleteTemplate(id string) error {
	return os.Remove(filepath.Join(tmplRoot(), sanitizeID(id)+".json"))
}

func BindTemplate(deviceID, templateID string, overlay map[string]any) error {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return fmt.Errorf("unknown device")
	}
	d.TemplateID = templateID
	if overlay != nil {
		d.Overlay = overlay
	}
	m[deviceID] = d
	return saveDevices(m)
}

func TemplateForDevice(deviceID string) (Template, error) {
	mu.Lock()
	d, ok := loadDevices()[deviceID]
	mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown device")
	}
	if d.Status != StatusApproved {
		return nil, fmt.Errorf("not approved")
	}
	tid := d.TemplateID
	if tid == "" {
		tid = "default"
	}
	t, err := GetTemplate(tid)
	if err != nil {
		t = Template{"id": tid, "role": "site"}
	}
	out := Template{}
	for k, v := range t {
		out[k] = v
	}
	if ov := d.Overlay; ov != nil {
		out["overlay"] = ov
		for _, sec := range []string{"network", "wifi", "vpn"} {
			base, _ := out[sec].(map[string]any)
			extra, _ := ov[sec].(map[string]any)
			if base == nil && extra == nil {
				continue
			}
			merged := map[string]any{}
			for k, v := range base {
				merged[k] = v
			}
			for k, v := range extra {
				merged[k] = v
			}
			out[sec] = merged
		}
		if v, ok := ov["lan_ip"]; ok {
			net, _ := out["network"].(map[string]any)
			if net == nil {
				net = map[string]any{}
			}
			net["lan_ip"] = v
			out["network"] = net
		}
		if v, ok := ov["ssid"]; ok {
			wifi, _ := out["wifi"].(map[string]any)
			if wifi == nil {
				wifi = map[string]any{}
			}
			wifi["ssid"] = v
			out["wifi"] = wifi
		}
	}
	out["device_id"] = deviceID
	return out, nil
}

func EnsureDefaultTemplate() {
	if _, err := GetTemplate("default"); err == nil {
		return
	}
	_ = SaveTemplate("default", Template{
		"guest": map[string]any{"enabled": false, "ssid": "Guest", "hidden": true},
		"id":   "default",
		"role": "site",
		"network": map[string]any{
			"lan_ip":   "192.168.50.1",
			"lan_mask": "255.255.255.0",
			"dhcp":     true,
		},
		"wifi": map[string]any{
			"ssid":       "Netductor",
			"encryption": "psk2",
			"key":        "",
		},
		"vpn": map[string]any{
			"enabled": true,
			"mode":    "socks",
		},
	})
}
