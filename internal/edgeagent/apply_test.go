package edgeagent

import "testing"

func TestDesiredUCIBandsInherit(t *testing.T) {
	lines := DesiredUCI(map[string]any{
		"wifi": map[string]any{"ssid_24": "Home", "key_24": "secret12"},
	})
	joined := ""
	for _, l := range lines {
		joined += l + "\n"
	}
	if !contains(joined, "default_radio0.ssid=Home") || !contains(joined, "default_radio1.ssid=Home") {
		t.Fatalf("inherit 5 from 24: %s", joined)
	}
	if !contains(joined, "default_radio1.key=secret12") {
		t.Fatalf("inherit key: %s", joined)
	}
}

func TestDesiredUCIPPPoE(t *testing.T) {
	lines := DesiredUCI(map[string]any{
		"network": map[string]any{
			"wan_proto": "pppoe", "pppoe_user": "u", "pppoe_pass": "p",
		},
	})
	joined := ""
	for _, l := range lines {
		joined += l + ";"
	}
	if !contains(joined, "network.wan.proto=pppoe") || !contains(joined, "username=u") {
		t.Fatal(joined)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || stringIndex(s, sub) >= 0)
}
func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
