package mikrotik

import (
	"strings"
	"testing"
)

func TestClientRSC(t *testing.T) {
	s := ClientRSC("mt-office", "1.2.3.4", "https://core.example", "netductor")
	if !strings.Contains(s, "mt-office") || !strings.Contains(s, "no VLESS") {
		t.Fatalf("rsc incomplete: %s", s)
	}
	if !strings.Contains(s, "1.2.3.4") {
		t.Fatalf("missing relay hint")
	}
}
