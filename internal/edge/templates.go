package edge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func tmplRoot() string {
	_ = ensure()
	d := filepath.Join(filepath.Dir(deviceDir("_")), "templates")
	_ = os.MkdirAll(d, 0o755)
	return d
}

type Template map[string]any

var tmplMu sync.Mutex // serializes SaveTemplate (read-modify-write); GetTemplate unlocked (atomic rename)

func sanitizeID(id string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, id)
}

func SaveTemplate(id string, t Template) error {
	tmplMu.Lock()
	defer tmplMu.Unlock()
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
		"guest": map[string]any{"enabled": false, "ssid": "Guest", "hidden": false},
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
			"enabled":       true,
			"mode":          "tun",
			"fallback":      "wan",
			"soft_fallback": true,
			"dns":           "vpn",
		},
	})
}


// applyVPNPolicyToMap merges validated policy keys from kvs into vpn map.
func applyVPNPolicyToMap(vpn map[string]any, kvs map[string]any) error {
	if vpn == nil {
		return fmt.Errorf("vpn map nil")
	}
	parseBool := func(v any) bool {
		switch x := v.(type) {
		case bool:
			return x
		case string:
			return x == "1" || strings.EqualFold(x, "true") || strings.EqualFold(x, "yes") || strings.EqualFold(x, "on")
		case float64:
			return x != 0
		default:
			return false
		}
	}
	inList := func(s string, allowed []string) bool {
		s = strings.ToLower(strings.TrimSpace(s))
		for _, a := range allowed {
			if s == a {
				return true
			}
		}
		return false
	}
	for k, v := range kvs {
		if k == "" {
			continue
		}
		switch k {
		case "enabled", "soft_fallback":
			vpn[k] = parseBool(v)
		case "dns", "dns_mode":
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
			if !inList(s, []string{"vpn", "wan", "off"}) {
				return fmt.Errorf("invalid dns %q (want vpn|wan|off)", s)
			}
			vpn["dns"] = s
			delete(vpn, "dns_mode")
		case "mode":
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
			if !inList(s, []string{"tun", "off", "socks", "mixed"}) {
				return fmt.Errorf("invalid mode %q (want tun|off|socks|mixed)", s)
			}
			vpn["mode"] = s
		case "fallback":
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
			if !inList(s, []string{"wan", "block", "none", "off"}) {
				return fmt.Errorf("invalid fallback %q (want wan|block)", s)
			}
			if s == "none" || s == "off" {
				s = "block"
			}
			vpn["fallback"] = s
		case "user", "subscription", "vless", "exit", "primary", "primary_host":
			// link / runtime fields — ignore on policy API
			continue
		default:
			return fmt.Errorf("unknown vpn key %q", k)
		}
	}
	return nil
}

// SetTemplateVPN merges vpn.* policy keys into template id (creates vpn section if missing).
func SetTemplateVPN(id string, kvs map[string]any) (Template, error) {
	EnsureDefaultTemplate()
	if id == "" {
		id = "default"
	}
	tmpl, err := GetTemplate(id)
	if err != nil {
		return nil, err
	}
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn == nil {
		vpn = map[string]any{}
	}
	if err := applyVPNPolicyToMap(vpn, kvs); err != nil {
		return nil, err
	}
	tmpl["vpn"] = vpn
	if err := SaveTemplate(id, tmpl); err != nil {
		return nil, err
	}
	return tmpl, nil
}

// MergeTemplate merges body into existing template (non-destructive).
// Nested map[string]any are shallow-merged; vpn keys are allowlisted.
// If replace is true, body fully replaces the template.
func MergeTemplate(id string, body Template, replace bool) (Template, error) {
	id = sanitizeID(id)
	if id == "" {
		return nil, fmt.Errorf("empty id")
	}
	if replace {
		if vm, ok := body["vpn"].(map[string]any); ok && vm != nil {
			// Validate policy keys even on full replace (link fields ignored by allowlist).
			probe := map[string]any{}
			if err := applyVPNPolicyToMap(probe, vm); err != nil {
				return nil, err
			}
			// Keep non-policy keys from body.vpn (e.g. future metadata) only if already validated path
			for k, v := range vm {
				if _, exists := probe[k]; exists {
					continue
				}
				switch k {
				case "user", "subscription", "vless", "exit", "primary", "primary_host":
					probe[k] = v
				}
			}
			body["vpn"] = probe
		}
		if err := SaveTemplate(id, body); err != nil {
			return nil, err
		}
		return GetTemplate(id)
	}
	EnsureDefaultTemplate()
	cur, err := GetTemplate(id)
	if err != nil {
		if err := SaveTemplate(id, body); err != nil {
			return nil, err
		}
		return GetTemplate(id)
	}
	for k, v := range body {
		if k == "id" {
			continue
		}
		if k == "vpn" {
			vm, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("vpn must be object")
			}
			base, _ := cur["vpn"].(map[string]any)
			if base == nil {
				base = map[string]any{}
			}
			if err := applyVPNPolicyToMap(base, vm); err != nil {
				return nil, err
			}
			cur["vpn"] = base
			continue
		}
		if vm, ok := v.(map[string]any); ok {
			base, _ := cur[k].(map[string]any)
			if base == nil {
				base = map[string]any{}
			}
			for sk, sv := range vm {
				base[sk] = sv
			}
			cur[k] = base
			continue
		}
		cur[k] = v
	}
	if err := SaveTemplate(id, cur); err != nil {
		return nil, err
	}
	return cur, nil
}
