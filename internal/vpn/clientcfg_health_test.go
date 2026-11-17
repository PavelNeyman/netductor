package vpn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOnlineSecondarysRequiresSingBoxOK(t *testing.T) {
	state := t.TempDir()
	etc := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", state)
	t.Setenv("NETDUCTOR_ETC", etc)
	dir := filepath.Join(state, "secondary")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	payload := map[string]any{
		"devices": []map[string]any{
			{
				"name": "bad", "public_ip": "1.1.1.1", "pbk": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				"sid": "abcd1234", "sni": "api.vk.me", "singbox_ok": false, "uplink_ok": true,
				"last_seen": now.Format(time.RFC3339Nano),
			},
			{
				"name": "nd-egress-ru-msk", "region": "ru-msk", "public_ip": "2.2.2.2",
				"pbk": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", "sid": "dcba4321",
				"sni": "api.vk.me", "singbox_ok": true, "uplink_ok": false,
				"last_seen": now.Format(time.RFC3339Nano),
			},
			{
				"name": "ok2", "public_ip": "3.3.3.3", "pbk": "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
				"sid": "11223344", "sni": "api.vk.me", "singbox_ok": true, "uplink_ok": true,
				"last_seen": now.Format(time.RFC3339Nano),
			},
		},
	}
	b, _ := json.Marshal(payload)
	if err := os.WriteFile(filepath.Join(dir, "devices.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(etc, "preferred_secondary_ip"), []byte("3.3.3.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := loadOnlineSecondarys()
	if len(out) != 2 {
		t.Fatalf("want 2 healthy, got %d %#v", len(out), out)
	}
	if out[0].PublicIP != "3.3.3.3" {
		t.Fatalf("preferred first: got %s", out[0].PublicIP)
	}
	for _, r := range out {
		if r.PublicIP == "1.1.1.1" {
			t.Fatal("singbox down must not appear in sub candidates")
		}
	}
}

func TestSetPreferredSecondaryIP(t *testing.T) {
	etc := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", etc)
	if err := SetPreferredSecondaryIP("9.9.9.9"); err != nil {
		t.Fatal(err)
	}
	if preferredSecondaryIP() != "9.9.9.9" {
		t.Fatal(preferredSecondaryIP())
	}
	if err := SetPreferredSecondaryIP(""); err != nil {
		t.Fatal(err)
	}
	if preferredSecondaryIP() != "" {
		t.Fatal("clear")
	}
}
