package svcpaths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyUserPathTargets(t *testing.T) {
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

func TestPrimaryTunnelAddrsStrictCorrupt(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	if err := os.WriteFile(filepath.Join(dir, "material.json"), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	sp, ps, err := PrimaryTunnelAddrsStrict()
	if err == nil {
		t.Fatal("expected error for corrupt material")
	}
	if sp != "" || ps != "" {
		t.Fatalf("corrupt must not return slot-0 addrs, got %s/%s", sp, ps)
	}
}

func TestPrimaryTunnelAddrsStrictIncomplete(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	b := []byte(`{"sp_pub_p":"x","addr_p_sp":"10.87.20.1","addr_p_ps":"10.87.21.1"}`)
	if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	sp, ps, err := PrimaryTunnelAddrsStrict()
	if err == nil {
		t.Fatal("expected error for incomplete material")
	}
	if sp != "" || ps != "" {
		t.Fatalf("incomplete must not return addrs, got %s/%s", sp, ps)
	}
}

func TestDetectRoleUsesHostRole(t *testing.T) {
	if detectRole() != hostRole() {
		t.Fatalf("detectRole=%q hostRole=%q", detectRole(), hostRole())
	}
}

func TestPrimaryTunnelAddrsStrictPartialAddrs(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	slot := SlotTable()[1]
	m := Material{
		SPPrivP: "x", SPPubP: "y", SPPrivS: "z", SPPubS: "w",
		PSPrivP: "a", PSPubP: "b", PSPrivS: "c", PSPubS: "d",
		AddrP_PS: slot.AddrP_PS,
		Slot:     1,
		PeerID:   "peer-partial",
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	sp, ps, err := PrimaryTunnelAddrsStrict()
	if err == nil {
		t.Fatal("expected error for partial addrs on slot 1")
	}
	if sp != "" || ps != "" {
		t.Fatalf("must not return slot-0, got %s/%s", sp, ps)
	}
}

func TestPrimaryTunnelAddrsStrictSlotMismatch(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	m := Material{
		SPPrivP: "x", SPPubP: "y", SPPrivS: "z", SPPubS: "w",
		PSPrivP: "a", PSPubP: "b", PSPrivS: "c", PSPubS: "d",
		AddrP_SP: AddrP_SP,
		AddrS_SP: AddrS_SP,
		AddrP_PS: AddrP_PS,
		AddrS_PS: AddrS_PS,
		Slot:     1,
		PeerID:   "peer-mismatch",
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := PrimaryTunnelAddrsStrict()
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestHealthTimerPrintfNewlineInSource(t *testing.T) {
	// installHealthTimer embeds a printf format ending with \n for the shell.
	// Read bootstrap source and ensure the format uses a single backslash-n (not \\n).
	src, err := os.ReadFile("bootstrap.go")
	if err != nil {
		t.Skip("bootstrap.go not readable from test cwd:", err)
	}
	needle := []byte(`note":"shell-fallback-peer"}`)
	i := indexBytes(src, needle)
	if i < 0 {
		t.Fatal("printf format not found in bootstrap.go")
	}
	// After the closing quote of the JSON, expect \n for printf (one 0x5c then 'n')
	rest := src[i+len(needle):]
	if len(rest) < 3 || rest[0] != '\\' || rest[1] != 'n' {
		t.Fatalf("expected \\n after JSON in printf format, got %q", rest[:min(10, len(rest))])
	}
	if len(rest) > 2 && rest[1] == '\\' {
		t.Fatal("double-backslash before n would print literal \\n")
	}
}

func indexBytes(b, sub []byte) int {
	for i := 0; i+len(sub) <= len(b); i++ {
		ok := true
		for j := range sub {
			if b[i+j] != sub[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestInstallHealthTimerRejectsCorruptMaterial(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	if err := os.WriteFile(filepath.Join(dir, "material.json"), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	// installHealthTimer writes under /usr/local and systemctl — skip if not root;
	// unit-test the addr resolution path via exported probe: call validate path only.
	// We check PrimaryTunnelAddrsStrict already fails; health timer must use same gate.
	_, _, _, _, err := healthTimerProbeAddrs()
	if err == nil {
		t.Fatal("corrupt material must refuse health timer addrs")
	}
	if os.IsNotExist(err) {
		t.Fatal("corrupt is not NotExist")
	}
}

func TestBootstrapRejectsInvalidBeforeSideEffects(t *testing.T) {
	m := &Material{Slot: 1, AddrP_SP: "10.87.20.1"} // incomplete slot 1
	err := BootstrapPrimaryWith(m, BootstrapPrimaryOpts{})
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("primary: %v", err)
	}
	if _, statErr := os.Stat("/etc/wireguard/nd-svc-sp1.conf"); statErr == nil {
		t.Fatal("primary wrote slot1 wg conf before identity fail")
	}
	sec := &Material{
		Slot: 1, AddrP_SP: "10.87.20.1",
		SPPrivS: "s", PSPrivS: "p", SPPubP: "a", PSPubP: "b",
	}
	err = BootstrapSecondary(sec)
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("secondary: %v", err)
	}
	if _, statErr := os.Stat("/etc/wireguard/nd-svc-sp1.conf"); statErr == nil {
		t.Fatal("secondary wrote slot1 wg conf before identity fail")
	}
}

func TestMaterialIfacesStrictFromVerifiedSlot(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	slot := SlotTable()[2]
	m := Material{
		SPPrivP: "x", SPPubP: "y", SPPrivS: "z", SPPubS: "w",
		PSPrivP: "a", PSPubP: "b", PSPrivS: "c", PSPubS: "d",
		AddrP_SP: slot.AddrP_SP, AddrS_SP: slot.AddrS_SP,
		AddrP_PS: slot.AddrP_PS, AddrS_PS: slot.AddrS_PS,
		// ifaces omitted — must come from SlotTable after identity OK
		Slot: 2, PeerID: "peer-if",
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	sp, ps, err := MaterialIfacesStrict()
	if err != nil {
		t.Fatal(err)
	}
	if sp != slot.IfaceSP || ps != slot.IfacePS {
		t.Fatalf("want %s/%s got %s/%s", slot.IfaceSP, slot.IfacePS, sp, ps)
	}
}

func TestResolveBootstrapParamsSlot1MissingPorts(t *testing.T) {
	slot := SlotTable()[1]
	m := &Material{
		Slot:     1,
		AddrP_SP: slot.AddrP_SP, AddrS_SP: slot.AddrS_SP,
		AddrP_PS: slot.AddrP_PS, AddrS_PS: slot.AddrS_PS,
	}
	_, _, _, _, portSP, portPS, wssSP, wssPS, ifaceSP, ifacePS, err := resolveBootstrapParams(m)
	if err != nil {
		t.Fatal(err)
	}
	if ifaceSP != slot.IfaceSP || ifacePS != slot.IfacePS || portSP != slot.PortSP || wssSP != slot.WSS_SP || portPS != slot.PortPS || wssPS != slot.WSS_PS {
		t.Fatalf("got iface %s/%s ports %d/%d wss %d/%d want slot1", ifaceSP, ifacePS, portSP, portPS, wssSP, wssPS)
	}
	if ifaceSP == "nd-svc-sp" || portSP == PortSP {
		t.Fatal("fell back to slot 0")
	}
}

func TestResolveBootstrapParamsRejectsForeignPort(t *testing.T) {
	slot := SlotTable()[1]
	m := &Material{
		Slot:     1,
		AddrP_SP: slot.AddrP_SP, AddrS_SP: slot.AddrS_SP,
		AddrP_PS: slot.AddrP_PS, AddrS_PS: slot.AddrS_PS,
		PortSP: PortSP, // slot 0
	}
	_, _, _, _, _, _, _, _, _, _, err := resolveBootstrapParams(m)
	if err == nil || !strings.Contains(err.Error(), "port_sp") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveBootstrapParamsPartialPortsFillFromSlot(t *testing.T) {
	slot := SlotTable()[2]
	m := &Material{
		Slot:     2,
		AddrP_SP: slot.AddrP_SP, AddrS_SP: slot.AddrS_SP,
		AddrP_PS: slot.AddrP_PS, AddrS_PS: slot.AddrS_PS,
		PortSP: slot.PortSP,
	}
	_, _, _, _, portSP, portPS, wssSP, _, ifaceSP, _, err := resolveBootstrapParams(m)
	if err != nil {
		t.Fatal(err)
	}
	if portSP != slot.PortSP || portPS != slot.PortPS || wssSP != slot.WSS_SP || ifaceSP != slot.IfaceSP {
		t.Fatalf("partial not filled from slot 2: %d %d %d %s", portSP, portPS, wssSP, ifaceSP)
	}
}

func TestResolveBootstrapParamsSlot0PortOnlyRejected(t *testing.T) {
	m := &Material{Slot: 0, PortSP: 51830}
	_, _, _, _, _, _, _, _, _, _, err := resolveBootstrapParams(m)
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("slot 0 with only PortSP must fail, got %v", err)
	}
}

func TestProbeHealthRejectsCorruptMaterial(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	if err := os.WriteFile(filepath.Join(dir, "material.json"), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := ProbeHealth()
	if err == nil || !strings.Contains(err.Error(), "material") {
		t.Fatalf("corrupt must not probe slot 0, got %v", err)
	}
}
