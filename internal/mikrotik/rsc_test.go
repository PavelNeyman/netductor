package mikrotik

import "testing"

func TestClientRSC(t *testing.T) {
	s := ClientRSC("mt-office", "1.2.3.4", "https://core.example", "netductor")
	if !stringsContains(s, "nd-heartbeat") || !stringsContains(s, "1.2.3.4") {
		t.Fatalf("rsc incomplete: %s", s)
	}
}

func stringsContains(s, sub string) bool {
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
