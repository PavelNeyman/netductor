package edge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetTemplateVPNAllowlist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	// paths may use different env — write via EnsureDefaultTemplate after chdir state
	_ = os.MkdirAll(filepath.Join(dir, "edge", "templates"), 0o755)
	EnsureDefaultTemplate()
	_, err := SetTemplateVPN("default", map[string]any{"dns": "nope"})
	if err == nil {
		t.Fatal("expected invalid dns")
	}
	tmpl, err := SetTemplateVPN("default", map[string]any{"dns": "wan", "fallback": "block", "soft_fallback": true})
	if err != nil {
		t.Fatal(err)
	}
	vpn, _ := tmpl["vpn"].(map[string]any)
	if vpn["dns"] != "wan" || vpn["fallback"] != "block" {
		t.Fatalf("vpn=%v", vpn)
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
