package edge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetTemplateVPNAllowlist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	_ = os.MkdirAll(filepath.Join(dir, "edge", "templates"), 0o755)
	EnsureDefaultTemplate()
	_, err := SetTemplateVPN("default", map[string]any{"dns": "nope"})
	if err == nil {
		t.Fatal("expected invalid dns")
	}
	_, err = SetTemplateVPN("default", map[string]any{"mode": "wireguard"})
	if err == nil {
		t.Fatal("expected invalid mode")
	}
	_, err = SetTemplateVPN("default", map[string]any{"unknown_key": "x"})
	if err == nil {
		t.Fatal("expected unknown key")
	}
	tmpl, err := SetTemplateVPN("default", map[string]any{
		"dns": "wan", "fallback": "block", "soft_fallback": true, "enabled": "yes", "mode": "tun",
	})
	if err != nil {
		t.Fatal(err)
	}
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn["dns"] != "wan" || vpn["fallback"] != "block" {
		t.Fatalf("vpn=%v", vpn)
	}
	if vpn["enabled"] != true {
		t.Fatalf("enabled want true got %v", vpn["enabled"])
	}
	// fallback none → block
	tmpl, err = SetTemplateVPN("default", map[string]any{"fallback": "none"})
	if err != nil {
		t.Fatal(err)
	}
	vpn, _ = tmpl["vpn"].(map[string]any)
	if vpn["fallback"] != "block" {
		t.Fatalf("fallback none→block got %v", vpn["fallback"])
	}
	// link fields ignored on policy API
	tmpl, err = SetTemplateVPN("default", map[string]any{"vless": "vless://should-ignore", "dns": "vpn"})
	if err != nil {
		t.Fatal(err)
	}
	vpn, _ = tmpl["vpn"].(map[string]any)
	if _, ok := vpn["vless"]; ok {
		t.Fatalf("vless must not be stored via SetTemplateVPN: %v", vpn)
	}
	if vpn["dns"] != "vpn" {
		t.Fatalf("dns=%v", vpn["dns"])
	}
}

func TestMergeTemplatePreservesSections(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	EnsureDefaultTemplate()
	_ = SaveTemplate("site", Template{
		"id": "site",
		"wifi": map[string]any{"ssid": "home"},
		"vpn":  map[string]any{"dns": "vpn", "mode": "tun"},
	})
	got, err := MergeTemplate("site", Template{"vpn": map[string]any{"dns": "wan"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	wifi, _ := got["wifi"].(map[string]any)
	if wifi["ssid"] != "home" {
		t.Fatalf("wifi wiped: %v", got)
	}
	vpn, _ := got["vpn"].(map[string]any)
	if vpn["dns"] != "wan" {
		t.Fatalf("dns not merged: %v", vpn)
	}
	if vpn["mode"] != "tun" {
		t.Fatalf("mode lost: %v", vpn)
	}
}

func TestMergeTemplateRejectsBadVPN(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	EnsureDefaultTemplate()
	_ = SaveTemplate("x", Template{"id": "x", "vpn": map[string]any{"dns": "vpn"}})
	_, err := MergeTemplate("x", Template{"vpn": map[string]any{"dns": "evil"}}, false)
	if err == nil {
		t.Fatal("expected reject")
	}
	// replace with bad vpn
	_, err = MergeTemplate("x", Template{"id": "x", "vpn": map[string]any{"mode": "nope"}}, true)
	if err == nil {
		t.Fatal("expected replace reject")
	}
	// replace valid — full body but only allowed vpn keys kept in policy path
	got, err := MergeTemplate("x", Template{
		"id": "x",
		"wifi": map[string]any{"ssid": "new"},
		"vpn":  map[string]any{"dns": "off", "mode": "off"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	wifi, _ := got["wifi"].(map[string]any)
	if wifi["ssid"] != "new" {
		t.Fatalf("%v", got)
	}
	vpn, _ := got["vpn"].(map[string]any)
	if vpn["dns"] != "off" || vpn["mode"] != "off" {
		t.Fatalf("%v", vpn)
	}
}

func TestTemplateWithVPNDisabledSkipsLinks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	EnsureDefaultTemplate()
	_ = SaveTemplate("default", Template{
		"id": "default",
		"vpn": map[string]any{
			"enabled": false, "dns": "wan", "fallback": "block", "mode": "tun",
		},
	})
	// bind to device
	_ = os.MkdirAll(filepath.Join(dir, "edge"), 0o700)
	t.Setenv("NETDUCTOR_EDGE_DIR", filepath.Join(dir, "edge"))
	Enroll(map[string]any{"device_id": "r1"})
	_, _ = Approve("r1")
	_ = BindTemplate("r1", "default", nil)
	tmpl, err := TemplateWithVPN("r1")
	if err != nil {
		t.Fatal(err)
	}
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn["dns"] != "wan" || vpn["fallback"] != "block" {
		t.Fatalf("policy changed: %v", vpn)
	}
	if _, ok := vpn["vless"]; ok {
		t.Fatalf("must not inject links when disabled: %v", vpn)
	}
}

func TestEdgeVPNUserLength(t *testing.T) {
	long := strings.Repeat("z", 120)
	name := edgeVPNUser(long)
	if len(name) > 64 {
		t.Fatalf("len=%d name=%q", len(name), name)
	}
	if !strings.HasPrefix(name, "edge-") {
		t.Fatalf("%q", name)
	}
	if !vpnValidNameish(name) {
		t.Fatalf("unexpected name %q", name)
	}
}

func vpnValidNameish(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for i, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if i == 0 && (r == '-' || r == '_') {
			// first may be letter/digit per ValidName
		}
		if !ok {
			return false
		}
	}
	return true
}
