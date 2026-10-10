package notify

import "testing"

func TestRecoveryMessageKnownKeys(t *testing.T) {
	if recoveryMessage("svcpath:sp") == "" {
		t.Fatal("sp")
	}
	if recoveryMessage("ok:svcpath:sp") != "" {
		t.Fatal("ok prefix")
	}
	if !contains(recoveryMessage("secondary:foo:uplink"), "uplink") {
		t.Fatal("uplink")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || 
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
