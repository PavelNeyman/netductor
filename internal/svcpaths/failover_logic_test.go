package svcpaths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyUserPathTargets(t *testing.T) {
	// Legacy slot-0 constants remain the fallback when material is absent.
	if AddrP_SP != "10.87.10.1" || AddrP_PS != "10.87.11.1" {
		t.Fatalf("peer addrs %s %s", AddrP_SP, AddrP_PS)
	}
}

func TestPrimaryTunnelAddrsSlotAware(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })

	for _, idx := range []int{1, 2, 3} {
		slot := SlotTable()[idx]
		m := Material{
			SPPrivP: "x", SPPubP: "y", SPPrivS: "z", SPPubS: "w",
			PSPrivP: "a", PSPubP: "b", PSPrivS: "c", PSPubS: "d",
			Primary: "1.2.3.4", Secondary: "5.6.7.8",
			AddrP_SP: slot.AddrP_SP, AddrS_SP: slot.AddrS_SP,
			AddrP_PS: slot.AddrP_PS, AddrS_PS: slot.AddrS_PS,
			PortSP: slot.PortSP, PortPS: slot.PortPS,
			WSS_SP: slot.WSS_SP, WSS_PS: slot.WSS_PS,
			IfaceSP: slot.IfaceSP, IfacePS: slot.IfacePS,
			Slot: idx, PeerID: "peer-test",
		}
		b, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
			t.Fatal(err)
		}
		sp, ps := PrimaryTunnelAddrs()
		if sp != slot.AddrP_SP || ps != slot.AddrP_PS {
			t.Fatalf("slot %d: want primary %s/%s got %s/%s", idx, slot.AddrP_SP, slot.AddrP_PS, sp, ps)
		}
		ssp, sps := SecondaryTunnelAddrs()
		if ssp != slot.AddrS_SP || sps != slot.AddrS_PS {
			t.Fatalf("slot %d: want secondary %s/%s got %s/%s", idx, slot.AddrS_SP, slot.AddrS_PS, ssp, sps)
		}
		is, ip := MaterialIfaces()
		if is != slot.IfaceSP || ip != slot.IfacePS {
			t.Fatalf("slot %d: ifaces want %s/%s got %s/%s", idx, slot.IfaceSP, slot.IfacePS, is, ip)
		}
	}
}

func TestPrimaryTunnelAddrsFallbackSlot0(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	// no material.json → constants
	sp, ps := PrimaryTunnelAddrs()
	if sp != AddrP_SP || ps != AddrP_PS {
		t.Fatalf("fallback want %s/%s got %s/%s", AddrP_SP, AddrP_PS, sp, ps)
	}
}

func TestDefaultPolicyStable(t *testing.T) {
	p := DefaultPolicy()
	if p.StableSeconds < 10 {
		t.Fatal(p.StableSeconds)
	}
}
