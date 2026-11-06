package svcpaths

import "testing"

// CF-3: slot 1 primary tunnel VIPs must not equal legacy slot 0.
func TestSlotTableDistinctFromSlot0(t *testing.T) {
	s0, s1 := SlotTable()[0], SlotTable()[1]
	if s1.AddrP_SP == s0.AddrP_SP || s1.AddrP_PS == s0.AddrP_PS {
		t.Fatalf("slot1 VIPs collide with slot0: %+v vs %+v", s1, s0)
	}
	if s1.AddrP_SP != "10.87.20.1" || s1.AddrP_PS != "10.87.21.1" {
		t.Fatalf("slot1 primary VIPs: %s %s", s1.AddrP_SP, s1.AddrP_PS)
	}
	if s1.IfaceSP == s0.IfaceSP {
		t.Fatalf("slot1 iface must differ: %s", s1.IfaceSP)
	}
}
