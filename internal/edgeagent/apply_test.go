package edgeagent

import (
	"strings"
	"testing"
)

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


func TestShellApplyStagedNoReload(t *testing.T) {
	s := ShellApplyStaged([]string{"network.lan.ipaddr=10.0.0.1"})
	if strings.Contains(s, "/etc/init.d/network reload") || strings.Contains(s, "wifi reload") {
		t.Fatalf("staged must not reload: %s", s)
	}
	if !strings.Contains(s, "uci commit") {
		t.Fatal(s)
	}
	// ash: unquoted ( ) in echo → syntax error: unexpected "("
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "echo ") && !strings.HasPrefix(line, "echo '") && !strings.HasPrefix(line, `echo "`) {
			if strings.Contains(line, "(") || strings.Contains(line, ")") {
				t.Fatalf("unquoted paren in remote echo (ash-unsafe): %s", line)
			}
		}
	}
	live := ShellApply([]string{"network.lan.ipaddr=10.0.0.1"})
	if !strings.Contains(live, "network reload") {
		t.Fatalf("live should reload: %s", live)
	}
}
