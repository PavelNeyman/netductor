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

func TestShadowrocketKeepsTwoProfilesSameHost(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	t.Setenv("NETDUCTOR_VPN_HOST", "vpn.example.test")
	path := paths.SecondaryDevicesFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	raw := `{"devices":[{"public_ip":"203.0.113.11","pbk":"pbk-a","sid":"sid-a","sni":"sni.example.test","last_seen":"` + now + `"},{"public_ip":"203.0.113.12","pbk":"pbk-b","sid":"sid-b","sni":"sni.example.test","last_seen":"` + now + `"}]}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := ShadowrocketJSON(ClientEndpoints{Name: "alice", UUID: "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "pbk-a") || !strings.Contains(s, "pbk-b") {
		t.Fatalf("same hostname dropped a profile: %s", s)
	}
}

func TestSubscriptionPreferredSecondaryFirstAfterPromote(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(etcDir, "vpn-users.json"), []byte(
		`{"users":[{"name":"alice","uuid":"00000000-0000-0000-0000-000000000001","enabled":true,"sub_profile":"secondary"}]}`,
	), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	type device struct {
		PublicIP string    `json:"public_ip"`
		PBK      string    `json:"pbk"`
		SID      string    `json:"sid"`
		SNI      string    `json:"sni"`
		LastSeen time.Time `json:"last_seen"`
	}
	path := paths.SecondaryDevicesFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(struct {
		Devices []device `json:"devices"`
	}{Devices: []device{
		{PublicIP: "203.0.113.11", PBK: "pbk-a", SID: "sid-a", SNI: "sni.example", LastSeen: now},
		{PublicIP: "203.0.113.22", PBK: "pbk-b", SID: "sid-b", SNI: "sni.example", LastSeen: now},
	}})
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	// Simulate peer promote: preferred B
	if err := os.WriteFile(filepath.Join(etcDir, "preferred_secondary_ip"), []byte("203.0.113.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	body, err := SubscriptionBody("alice")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 profiles, got %d: %q", len(lines), body)
	}
	if !strings.Contains(lines[0], "pbk-b") {
		t.Fatalf("promoted B must be first profile: %q", lines[0])
	}
	if !strings.Contains(lines[1], "pbk-a") {
		t.Fatalf("A should remain second: %q", lines[1])
	}
	if !strings.Contains(body, "@vpn.example.test:443") {
		t.Fatalf("must use shared hostname: %q", body)
	}
	// Empty SID omitted
	raw2, _ := json.Marshal(struct {
		Devices []device `json:"devices"`
	}{Devices: []device{
		{PublicIP: "203.0.113.22", PBK: "pbk-b", SID: "sid-b", SNI: "sni.example", LastSeen: now},
		{PublicIP: "203.0.113.33", PBK: "pbk-empty", SID: "", SNI: "sni.example", LastSeen: now},
	}})
	_ = os.WriteFile(path, raw2, 0o600)
	body, err = SubscriptionBody("alice")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "pbk-empty") {
		t.Fatalf("empty SID must be omitted: %q", body)
	}
	if !strings.Contains(body, "pbk-b") {
		t.Fatalf("B must remain: %q", body)
	}
}

func TestSubscriptionOmitsEmptyPBK(t *testing.T) {
	root := t.TempDir()
	etcDir := filepath.Join(root, "etc")
	t.Setenv("NETDUCTOR_ETC", etcDir)
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	t.Setenv("NETDUCTOR_VPN_HOST", "vpn.example.test")
	_ = os.MkdirAll(filepath.Join(etcDir, "secrets"), 0o700)
	_ = os.WriteFile(filepath.Join(etcDir, "secrets", "singbox_reality_public"), []byte("core-pbk"), 0o600)
	_ = os.WriteFile(filepath.Join(etcDir, "secrets", "singbox_short_id"), []byte("core-sid"), 0o600)
	_ = os.WriteFile(filepath.Join(etcDir, "vpn-users.json"), []byte(
		`{"users":[{"name":"bob","uuid":"00000000-0000-0000-0000-000000000002","enabled":true,"sub_profile":"secondary"}]}`,
	), 0o600)
	path := paths.SecondaryDevicesFile()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	now := time.Now().UTC()
	raw, _ := json.Marshal(map[string]any{"devices": []map[string]any{
		{"public_ip": "203.0.113.9", "pbk": "", "sid": "sid-invalid", "sni": "x", "last_seen": now},
		{"public_ip": "203.0.113.8", "pbk": "   ", "sid": "sid-ws", "sni": "x", "last_seen": now},
		{"public_ip": "203.0.113.7", "pbk": "good-pbk", "sid": "   ", "sni": "x", "last_seen": now},
		{"public_ip": "203.0.113.6", "pbk": "keep-pbk", "sid": "keep-sid", "sni": "x", "last_seen": now},
	}})
	_ = os.WriteFile(path, raw, 0o600)
	body, err := SubscriptionBody("bob")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"sid-invalid", "sid-ws", "good-pbk", "203.0.113.9", "203.0.113.8", "203.0.113.7"} {
		if strings.Contains(body, bad) {
			t.Fatalf("must omit invalid secondary marker %q in %q", bad, body)
		}
	}
	if !strings.Contains(body, "keep-pbk") || !strings.Contains(body, "keep-sid") {
		t.Fatalf("valid secondary missing: %q", body)
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 1 {
		t.Fatalf("want exactly 1 profile, got %d: %q", len(lines), body)
	}
}
