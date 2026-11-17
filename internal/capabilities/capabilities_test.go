package capabilities

import "testing"

func TestSetHas(t *testing.T) {
	m := Set(nil, VPNIngress, true)
	if !Has(m, VPNIngress) {
		t.Fatal("has")
	}
	m = Set(m, VPNIngress, false)
	if Has(m, VPNIngress) {
		t.Fatal("cleared")
	}
}
