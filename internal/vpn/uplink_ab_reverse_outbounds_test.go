package vpn

import (
	"os"
	"path/filepath"
	"testing"
)

// When forward AB profiles are enabled, reverse outbounds must still be generated.
// Regression: appendUplinkABOutbounds returned only forward tags while
// uplinkABRouteRules still emitted reverse rules → missing outbound for nd-up-rev-*.
func TestAppendUplinkABOutboundsIncludesReverseWhenForwardPresent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir)
	t.Setenv(uplinkABEnv, "1")
	t.Setenv(uplinkABUsersEnv, "Pavel")
	_ = os.MkdirAll(dir, 0o755)

	cfg := UplinkABConfig{
		Enabled:         true,
		ReverseEnabled:  true,
		Users:           []string{"Pavel"},
		Profiles:        []string{"ws", "nomux"},
		ReverseProfiles: []string{"ws", "nomux", "grpc", "anytls"},
		PrimaryPublicIP: "2.27.118.70",
		BasePort:        9444,
		ReverseBasePort: 9544,
	}
	if err := saveUplinkABConfig(cfg); err != nil {
		t.Fatal(err)
	}
	// sanity: file under NETDUCTOR_ETC
	if _, err := os.Stat(filepath.Join(dir, "uplink_ab.json")); err != nil {
		t.Fatal(err)
	}

	b := &SecondaryBundle{
		CoreIP:     "2.27.118.70",
		CoreSNI:    "api.vk.me",
		CoreVless:  443,
		UplinkUUID: "00000000-0000-0000-0000-000000000001",
	}
	out := appendUplinkABOutbounds(nil, b)
	tags := map[string]bool{}
	for _, o := range out {
		m, ok := o.(map[string]any)
		if !ok {
			continue
		}
		if tag, _ := m["tag"].(string); tag != "" {
			tags[tag] = true
		}
	}
	for _, need := range []string{
		"uplink-ab-ws",
		"uplink-ab-nomux",
		"uplink-ab-rev-ws",
		"uplink-ab-rev-nomux",
		"uplink-ab-rev-grpc",
		"uplink-ab-rev-anytls",
	} {
		if !tags[need] {
			t.Fatalf("missing outbound tag %s; got %v", need, tags)
		}
	}
	// Local reverse targets (revbridge)
	for _, o := range out {
		m, ok := o.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := m["tag"].(string)
		if tag == "uplink-ab-rev-ws" {
			if m["server"] != "127.0.0.1" {
				t.Fatalf("rev-ws server want 127.0.0.1 got %v", m["server"])
			}
			port, _ := m["server_port"].(int)
			if port != 12543 {
				t.Fatalf("rev-ws local port want 12543 got %v", m["server_port"])
			}
		}
	}
}
