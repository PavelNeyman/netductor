package svcpaths

import "testing"

func TestValidatePeerID(t *testing.T) {
	if err := validatePeerID("peer-b"); err != nil {
		t.Fatal(err)
	}
	if err := validatePeerID("../evil"); err == nil {
		t.Fatal("expected invalid")
	}
	if err := validatePublicIP("203.0.113.1"); err != nil {
		t.Fatal(err)
	}
	if err := validatePublicIP("not-an-ip"); err == nil {
		t.Fatal("expected invalid ip")
	}
}

func TestSlotTableUniqueWSS(t *testing.T) {
	seen := map[int]bool{}
	for _, s := range SlotTable() {
		if seen[s.WSS_SP] {
			t.Fatalf("duplicate WSS_SP %d", s.WSS_SP)
		}
		seen[s.WSS_SP] = true
	}
}
