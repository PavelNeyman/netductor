package svcpaths

import "testing"

func TestApplyUserPathTargets(t *testing.T) {
	// Document contract: SP→10.87.10.1 PS→10.87.11.1 (constants).
	if AddrP_SP != "10.87.10.1" || AddrP_PS != "10.87.11.1" {
		t.Fatalf("peer addrs %s %s", AddrP_SP, AddrP_PS)
	}
}

func TestDefaultPolicyStable(t *testing.T) {
	p := DefaultPolicy()
	if p.StableSeconds < 10 {
		t.Fatal(p.StableSeconds)
	}
}
