package svcpaths

import "testing"

func TestPrimaryWSSUnitsSlotAware(t *testing.T) {
	a, b := primaryWSSUnits(IfaceSP, IfacePS)
	if a != "nd-wss-sp-server" || b != "nd-wss-ps-client" {
		t.Fatalf("slot0 units: %s %s", a, b)
	}
	a, b = primaryWSSUnits("nd-svc-sp1", "nd-svc-ps1")
	if a != "nd-wss-sp-server-nd-svc-sp1" || b != "nd-wss-ps-client-nd-svc-ps1" {
		t.Fatalf("slot1 units: %s %s", a, b)
	}
}
