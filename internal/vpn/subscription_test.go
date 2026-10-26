package vpn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestSubscriptionBodyIncludesAllSecondariesAndDropsRemovedPeer(t *testing.T) {
	root := t.TempDir()
	etcDir := filepath.Join(root, "etc")
	stateDir := filepath.Join(root, "state")
	t.Setenv("NETDUCTOR_ETC", etcDir)
	t.Setenv("NETDUCTOR_STATE", stateDir)
	t.Setenv("NETDUCTOR_VPN_HOST", "vpn.example.test")
	t.Setenv("NETDUCTOR_CORE_HOST", "core.example.test")
	t.Setenv("PUBLIC_IP", "198.51.100.10")

	if err := os.MkdirAll(filepath.Join(etcDir, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"singbox_reality_public": "core-pbk",
		"singbox_short_id":       "core-sid",
	} {
		if err := os.WriteFile(filepath.Join(etcDir, "secrets", name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	userRegistry := `{"users":[{"name":"alice","uuid":"00000000-0000-0000-0000-000000000001","enabled":true,"sub_profile":"secondary"}]}`
	if err := os.WriteFile(filepath.Join(etcDir, "vpn-users.json"), []byte(userRegistry), 0o600); err != nil {
		t.Fatal(err)
	}

	type device struct {
		PublicIP string    `json:"public_ip"`
		PBK      string    `json:"pbk"`
		SID      string    `json:"sid"`
		SNI      string    `json:"sni"`
		LastSeen time.Time `json:"last_seen"`
	}
	writeDevices := func(devices ...device) {
		t.Helper()
		path := paths.SecondaryDevicesFile()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(struct {
			Devices []device `json:"devices"`
		}{Devices: devices})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Now().UTC()
	a := device{PublicIP: "203.0.113.11", PBK: "secondary-a-pbk", SID: "a1", SNI: "sni.example.test", LastSeen: now}
	b := device{PublicIP: "203.0.113.22", PBK: "secondary-b-pbk", SID: "b2", SNI: "sni.example.test", LastSeen: now}
	writeDevices(a, b)

	body, err := SubscriptionBody("alice")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two Secondary profiles, got %d: %q", len(lines), body)
	}
	if !strings.Contains(body, "@vpn.example.test:443") {
		t.Fatalf("profiles must use shared DNS hostname: %q", body)
	}
	for _, want := range []string{"secondary-a-pbk", "secondary-b-pbk", "#nd-secondary-203-0-113-11", "#nd-secondary-203-0-113-22"} {
		if !strings.Contains(body, want) {
			t.Fatalf("subscription missing %q: %q", want, body)
		}
	}
	if strings.Contains(body, "core-pbk") {
		t.Fatalf("secondary profile should not include core: %q", body)
	}

	a.LastSeen = now.Add(-4 * time.Minute)
	writeDevices(a, b)
	body, err = SubscriptionBody("alice")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "secondary-a-pbk") || !strings.Contains(body, "secondary-b-pbk") {
		t.Fatalf("stale Secondary should disappear: %q", body)
	}

	a.LastSeen = now
	writeDevices(a, b)
	writeDevices(b)
	body, err = SubscriptionBody("alice")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "secondary-a-pbk") || !strings.Contains(body, "secondary-b-pbk") {
		t.Fatalf("removed Secondary should disappear: %q", body)
	}
}

func TestUniqueProfileLabelOnSameIP(t *testing.T) {
	used := map[string]int{}
	a := uniqueProfileLabel("nd-secondary-203-0-113-11", used)
	b := uniqueProfileLabel("nd-secondary-203-0-113-11", used)
	if a == b {
		t.Fatalf("labels collided: %s %s", a, b)
	}
	if a != "nd-secondary-203-0-113-11" || b != "nd-secondary-203-0-113-11-2" {
		t.Fatalf("got %s %s", a, b)
	}
}

func TestEmptySIDOmitted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	t.Setenv("NETDUCTOR_VPN_HOST", "vpn.example.test")
	if err := os.MkdirAll(filepath.Join(root, "etc", "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"singbox_reality_public": "core-pbk", "singbox_short_id": "core-sid"} {
		if err := os.WriteFile(filepath.Join(root, "etc", "secrets", name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "vpn-users.json"), []byte(`{"users":[{"name":"alice","uuid":"00000000-0000-0000-0000-000000000001","enabled":true,"sub_profile":"secondary"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	path := paths.SecondaryDevicesFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	raw := `{"devices":[{"public_ip":"203.0.113.11","pbk":"secondary-a-pbk","sid":"","sni":"sni.example.test","last_seen":"` + time.Now().UTC().Format(time.RFC3339) + `"}]}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	body, err := SubscriptionBody("alice")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "secondary-a-pbk") {
		t.Fatalf("empty SID must not produce a profile: %s", body)
	}
}
