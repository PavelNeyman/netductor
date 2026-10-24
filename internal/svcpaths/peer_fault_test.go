package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	preferredIPPath = filepath.Join(dir, "preferred_secondary_ip")
	preferredIPSecretsPath = filepath.Join(dir, "secrets", "preferred_secondary_ip")
	_ = os.MkdirAll(filepath.Dir(preferredIPSecretsPath), 0o700)
	t.Cleanup(func() {
		SetKeysRootForTest("")
		bootstrapPrimaryFn = BootstrapPrimaryWith
		savePeersIndexFn = SavePeersIndex
		writePeerOpFn = writePeerOpDefault
		generateMaterialFn = GenerateMaterial
		preferredIPPath = "/etc/netductor/preferred_secondary_ip"
		preferredIPSecretsPath = "/etc/netductor/secrets/preferred_secondary_ip"
		writeFileFn = os.WriteFile
	})
	return dir
}

func sampleMaterial(id, ip string, slot PeerSlot) *Material {
	m := &Material{
		SPPrivP: "sp_priv_p", SPPubP: "sp_pub_p",
		SPPrivS: "sp_priv_s", SPPubS: "sp_pub_s",
		PSPrivP: "ps_priv_p", PSPubP: "ps_pub_p",
		PSPrivS: "ps_priv_s", PSPubS: "ps_pub_s",
		Primary: "1.1.1.1", Secondary: ip, PeerID: id,
	}
	applySlotToMaterial(m, slot)
	return m
}

func writePeerMaterial(t *testing.T, m *Material) {
	t.Helper()
	dir := filepath.Join(peersDirPath(), m.PeerID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(peerMaterialPath(m.PeerID), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMaterialStrictCorruptRefuses(t *testing.T) {
	withTestRoot(t)
	slot := SlotTable()[0]
	m := sampleMaterial("p1", "203.0.113.1", slot)
	m.SPPrivP = ""
	writePeerMaterial(t, m)
	_, err := loadPeerMaterial("p1")
	if err == nil {
		t.Fatal("expected incomplete material error")
	}
}

func TestAllocateSlotsNoOverlap(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1}
	used := map[int]bool{}
	addrs := map[string]bool{}
	for i := 0; i < len(SlotTable()); i++ {
		s, err := allocateSlot(idx)
		if err != nil {
			t.Fatal(err)
		}
		if used[s.Index] {
			t.Fatalf("slot %d reused", s.Index)
		}
		used[s.Index] = true
		for _, a := range []string{s.AddrP_SP, s.AddrS_SP, s.AddrP_PS, s.AddrS_PS} {
			if addrs[a] {
				t.Fatalf("addr overlap %s", a)
			}
			addrs[a] = true
		}
		idx.Peers = append(idx.Peers, ServicePeer{ID: "x", State: "prepared", Slot: s})
	}
	if _, err := allocateSlot(idx); err == nil {
		t.Fatal("expected no free slots")
	}
}

func TestSlotWSSIsolation(t *testing.T) {
	wssSP, wssPS := map[int]bool{}, map[int]bool{}
	for _, s := range SlotTable() {
		if wssSP[s.WSS_SP] {
			t.Fatalf("duplicate WSS_SP %d", s.WSS_SP)
		}
		if wssPS[s.WSS_PS] {
			t.Fatalf("duplicate WSS_PS %d", s.WSS_PS)
		}
		wssSP[s.WSS_SP] = true
		wssPS[s.WSS_PS] = true
	}
}

func TestRecoverAfterMaterialBeforeIndex(t *testing.T) {
	withTestRoot(t)
	bootCalls := 0
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		bootCalls++
		return nil
	}
	slot := SlotTable()[1]
	m := sampleMaterial("peer-b", "203.0.113.10", slot)
	writePeerMaterial(t, m)
	_ = writePeerOp(peerOp{Op: "add", PeerID: "peer-b", PublicIP: "203.0.113.10", PrimaryIP: "1.1.1.1", SlotIndex: 1, Phase: "material"})

	if _, err := RecoverIncompletePeerAdd("peer-b"); err != nil {
		t.Fatal(err)
	}
	if bootCalls != 1 {
		t.Fatalf("bootstrap calls %d", bootCalls)
	}
	idx, _ := LoadPeersIndex()
	if idx.Find("peer-b") == nil {
		t.Fatal("not indexed")
	}
	if _, err := RecoverIncompletePeerAdd("peer-b"); err != nil {
		t.Fatal(err)
	}
	if bootCalls != 1 {
		t.Fatalf("second recover re-bootstrapped: %d", bootCalls)
	}
}

func TestRecoverMissingMaterialFails(t *testing.T) {
	withTestRoot(t)
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		t.Fatal("should not bootstrap")
		return nil
	}
	_ = writePeerOp(peerOp{Op: "add", PeerID: "ghost", PublicIP: "203.0.113.9", Phase: "material"})
	_, err := RecoverIncompletePeerAdd("ghost")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRecoverBootstrapFailedRetriesSameKeys(t *testing.T) {
	withTestRoot(t)
	slot := SlotTable()[0]
	m := sampleMaterial("peer-a", "203.0.113.5", slot)
	writePeerMaterial(t, m)
	privBefore := m.SPPrivP
	_ = writePeerOp(peerOp{Op: "add", PeerID: "peer-a", PublicIP: "203.0.113.5", SlotIndex: 0, Phase: "bootstrap_failed"})
	var sawPriv string
	bootstrapPrimaryFn = func(mm *Material, o BootstrapPrimaryOpts) error {
		sawPriv = mm.SPPrivP
		return nil
	}
	if _, err := RecoverIncompletePeerAdd("peer-a"); err != nil {
		t.Fatal(err)
	}
	if sawPriv != privBefore {
		t.Fatalf("keys regenerated")
	}
}

func TestPeerAddJournalFailureStopsBeforeBootstrap(t *testing.T) {
	withTestRoot(t)
	bootCalls, genCalls := 0, 0
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		bootCalls++
		return nil
	}
	generateMaterialFn = func(primaryIP, secondaryIP string) (*Material, error) {
		genCalls++
		return sampleMaterial("peer-x", secondaryIP, SlotTable()[0]), nil
	}
	writePeerOpFn = func(op peerOp) error {
		return fmt.Errorf("journal disk full")
	}
	_, _, err := PeerAdd(PeerAddOpts{PeerID: "peer-x", PublicIP: "203.0.113.20", PrimaryIP: "1.1.1.1"})
	if err == nil {
		t.Fatal("expected journal error")
	}
	if genCalls != 0 || bootCalls != 0 {
		t.Fatalf("gen=%d boot=%d want 0", genCalls, bootCalls)
	}
	idx, _ := LoadPeersIndex()
	if idx.Find("peer-x") != nil {
		t.Fatal("peer should not be indexed")
	}
}

func TestRecoverAfterIndexSaveFailure(t *testing.T) {
	withTestRoot(t)
	bootCalls := 0
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		bootCalls++
		return nil
	}
	slot := SlotTable()[1]
	m := sampleMaterial("peer-b", "203.0.113.10", slot)
	writePeerMaterial(t, m)
	_ = writePeerOp(peerOp{Op: "add", PeerID: "peer-b", PublicIP: "203.0.113.10", PrimaryIP: "1.1.1.1", SlotIndex: 1, Phase: "material"})

	savePeersIndexFn = func(idx *PeersIndex) error {
		return fmt.Errorf("disk full")
	}
	_, err := RecoverIncompletePeerAdd("peer-b")
	if err == nil {
		t.Fatal("expected index save error")
	}
	op, err := loadPeerOp("peer-b")
	if err != nil {
		t.Fatal(err)
	}
	if op.Phase != "index_failed" {
		t.Fatalf("phase %s want index_failed", op.Phase)
	}
	if bootCalls != 1 {
		t.Fatalf("boot %d", bootCalls)
	}

	// Second recover: phase index_failed → no second bootstrap
	savePeersIndexFn = SavePeersIndex
	if _, err := RecoverIncompletePeerAdd("peer-b"); err != nil {
		t.Fatal(err)
	}
	if bootCalls != 1 {
		t.Fatalf("bootstrap repeated: %d", bootCalls)
	}
	idx, _ := LoadPeersIndex()
	if idx.Find("peer-b") == nil {
		t.Fatal("not prepared")
	}
	if _, err := os.Stat(peerOpPath("peer-b")); !os.IsNotExist(err) {
		t.Fatal("op.json should be cleared")
	}
}

func TestRecoverCorruptOpJSON(t *testing.T) {
	withTestRoot(t)
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		t.Fatal("no bootstrap")
		return nil
	}
	_ = os.MkdirAll(filepath.Join(peersDirPath(), "bad"), 0o700)
	_ = os.WriteFile(peerOpPath("bad"), []byte("{not json"), 0o600)
	writePeerMaterial(t, sampleMaterial("bad", "203.0.113.8", SlotTable()[0]))
	_, err := RecoverIncompletePeerAdd("bad")
	if err == nil {
		t.Fatal("expected corrupt error")
	}
}

func TestRecoverUnknownPhase(t *testing.T) {
	withTestRoot(t)
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		t.Fatal("no bootstrap")
		return nil
	}
	m := sampleMaterial("p", "203.0.113.7", SlotTable()[0])
	writePeerMaterial(t, m)
	_ = writePeerOp(peerOp{Op: "add", PeerID: "p", PublicIP: "203.0.113.7", SlotIndex: 0, Phase: "weird"})
	_, err := RecoverIncompletePeerAdd("p")
	if err == nil {
		t.Fatal("expected unknown phase")
	}
}

func TestPeerPromotePreparedAndReady(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "old", PublicIP: "203.0.113.1", State: "active", Slot: SlotTable()[0]},
		{ID: "p1", PublicIP: "203.0.113.2", State: "prepared", Slot: SlotTable()[1]},
	}, ActiveID: "old"}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	if err := PeerPromote("p1"); err == nil {
		t.Fatal("promote prepared must fail")
	}
	idx.Peers[1].State = "ready"
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	if err := PeerPromote("p1"); err != nil {
		t.Fatal(err)
	}
	idx2, _ := LoadPeersIndex()
	if idx2.ActiveID != "p1" {
		t.Fatalf("active %s", idx2.ActiveID)
	}
	if idx2.Find("old").State != "ready" || idx2.Find("p1").State != "active" {
		t.Fatalf("%+v", idx2.Peers)
	}
	b, err := os.ReadFile(preferredIPPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "203.0.113.2\n" {
		t.Fatalf("preferred %q", b)
	}
	ab, _ := os.ReadFile(activePeerPathFile())
	if string(ab) != "p1\n" {
		t.Fatalf("active file %q", ab)
	}
}

func TestTwoPeersDifferentSlots(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1}
	s0, _ := allocateSlot(idx)
	idx.Peers = append(idx.Peers, ServicePeer{ID: "a", State: "active", Slot: s0})
	s1, _ := allocateSlot(idx)
	if s0.Index == s1.Index || s0.AddrS_SP == s1.AddrS_SP || s0.WSS_SP == s1.WSS_SP {
		t.Fatal("A/B not isolated")
	}
}

func TestRecoverInvalidSlotRefused(t *testing.T) {
	withTestRoot(t)
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		t.Fatal("no bootstrap")
		return nil
	}
	m := sampleMaterial("p", "203.0.113.7", SlotTable()[0])
	m.Slot = 999
	writePeerMaterial(t, m)
	_ = writePeerOp(peerOp{Op: "add", PeerID: "p", PublicIP: "203.0.113.7", SlotIndex: 999, Phase: "material"})
	_, err := RecoverIncompletePeerAdd("p")
	if err == nil {
		t.Fatal("expected invalid slot")
	}
}

func TestRecoverIndexFailedJournalErrorJoined(t *testing.T) {
	withTestRoot(t)
	bootCalls := 0
	bootstrapPrimaryFn = func(m *Material, o BootstrapPrimaryOpts) error {
		bootCalls++
		return nil
	}
	slot := SlotTable()[1]
	m := sampleMaterial("peer-b", "203.0.113.10", slot)
	writePeerMaterial(t, m)
	_ = writePeerOp(peerOp{Op: "add", PeerID: "peer-b", PublicIP: "203.0.113.10", PrimaryIP: "1.1.1.1", SlotIndex: 1, Phase: "material"})
	savePeersIndexFn = func(idx *PeersIndex) error {
		return fmt.Errorf("disk full")
	}
	writePeerOpFn = func(op peerOp) error {
		if op.Phase == "index_failed" {
			return fmt.Errorf("journal full")
		}
		return writePeerOpDefault(op)
	}
	_, err := RecoverIncompletePeerAdd("peer-b")
	if err == nil {
		t.Fatal("expected error")
	}
	es := err.Error()
	if !strings.Contains(es, "disk full") || !strings.Contains(es, "journal full") {
		t.Fatalf("want both errors: %v", err)
	}
	if bootCalls != 1 {
		t.Fatalf("boot %d", bootCalls)
	}
}

func Disabled_TestPeerPromotePreferredFailRollsBack(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "old", PublicIP: "203.0.113.1", State: "active", Slot: SlotTable()[0]},
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[1]},
	}, ActiveID: "old"}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(preferredIPPath, []byte("203.0.113.1\n"), 0o644)
	// make preferred path a directory so WriteFile fails
	_ = os.Remove(preferredIPPath)
	if err := os.Mkdir(preferredIPPath, 0o700); err != nil {
		t.Fatal(err)
	}
	err := PeerPromote("p1")
	if err == nil {
		t.Fatal("expected preferred write fail")
	}
	idx2, _ := LoadPeersIndex()
	if idx2.ActiveID != "old" {
		t.Fatalf("rollback active want old got %s", idx2.ActiveID)
	}
	if idx2.Find("old").State != "active" {
		t.Fatalf("old should be active again")
	}
}

func TestPeerPromoteSecretsFileWritten(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[0]},
	}}
	_ = SavePeersIndex(idx)
	if err := PeerPromote("p1"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(preferredIPSecretsPath)
	if err != nil || string(b) != "203.0.113.2\n" {
		t.Fatalf("secrets preferred %v %q", err, b)
	}
}


func Disabled_TestPeerPromoteActiveFileFailRollsBack(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "old", PublicIP: "203.0.113.1", State: "active", Slot: SlotTable()[0]},
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[1]},
	}, ActiveID: "old"}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(activePeerPathFile(), []byte("old\n"), 0o644)
	// replace active path with a directory so WriteFile fails after SavePeersIndex
	path := activePeerPathFile()
	_ = os.Remove(path)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := PeerPromote("p1"); err == nil {
		t.Fatal("expected fail")
	}
	idx2, _ := LoadPeersIndex()
	if idx2.ActiveID != "old" {
		t.Fatalf("ActiveID %s", idx2.ActiveID)
	}
	if idx2.Find("old").State != "active" || idx2.Find("p1").State != "ready" {
		t.Fatalf("states %+v", idx2.Peers)
	}
}

func Disabled_TestPeerPromoteClearsPreferredWhenNoneBefore(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[0]},
	}}
	_ = SavePeersIndex(idx)
	// no preferred files before
	// force secrets write fail after preferred wrote
	_ = os.MkdirAll(filepath.Dir(preferredIPSecretsPath), 0o700)
	sec := preferredIPSecretsPath
	_ = os.Remove(sec)
	if err := os.Mkdir(sec, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := PeerPromote("p1"); err == nil {
		t.Fatal("expected fail")
	}
	if _, err := os.Stat(preferredIPPath); !os.IsNotExist(err) {
		// preferred should be removed because it did not exist before
		b, _ := os.ReadFile(preferredIPPath)
		t.Fatalf("preferred should be absent, got %q err=%v", b, err)
	}
}


func TestSnapshotRefusesUnreadablePath(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[0]},
	}}
	_ = SavePeersIndex(idx)
	// directory at preferred path → snapshot error, no promote side effects
	if err := os.Mkdir(preferredIPPath, 0o700); err != nil {
		t.Fatal(err)
	}
	err := PeerPromote("p1")
	if err == nil {
		t.Fatal("expected snapshot error")
	}
	if !strings.Contains(err.Error(), "directory") && !strings.Contains(err.Error(), "snapshot") {
		t.Fatalf("unexpected: %v", err)
	}
	idx2, _ := LoadPeersIndex()
	if idx2.Find("p1").State != "ready" || idx2.ActiveID != "" {
		t.Fatalf("must not change registry: %+v", idx2)
	}
}

func TestPeerPromoteActiveWriteFailFullRollback(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "old", PublicIP: "203.0.113.1", State: "active", Slot: SlotTable()[0]},
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[1]},
	}, ActiveID: "old"}
	_ = SavePeersIndex(idx)
	_ = os.WriteFile(activePeerPathFile(), []byte("old\n"), 0o644)
	_ = os.WriteFile(preferredIPPath, []byte("203.0.113.1\n"), 0o644)
	_ = os.MkdirAll(filepath.Dir(preferredIPSecretsPath), 0o700)
	_ = os.WriteFile(preferredIPSecretsPath, []byte("203.0.113.1\n"), 0o600)

	failOnce := true
	writeFileFn = func(name string, data []byte, perm os.FileMode) error {
		if failOnce && name == activePeerPathFile() {
			failOnce = false
			return fmt.Errorf("active write denied")
		}
		return os.WriteFile(name, data, perm)
	}
	err := PeerPromote("p1")
	if err == nil {
		t.Fatal("expected fail")
	}
	if !strings.Contains(err.Error(), "active write denied") {
		t.Fatalf("%v", err)
	}
	if strings.Contains(err.Error(), "rollback:") {
		t.Fatalf("rollback should succeed, got %v", err)
	}
	idx2, _ := LoadPeersIndex()
	if idx2.ActiveID != "old" || idx2.Find("old").State != "active" || idx2.Find("p1").State != "ready" {
		t.Fatalf("%+v", idx2)
	}
	b, _ := os.ReadFile(activePeerPathFile())
	if string(b) != "old\n" {
		t.Fatalf("active file %q", b)
	}
	b, _ = os.ReadFile(preferredIPPath)
	if string(b) != "203.0.113.1\n" {
		t.Fatalf("preferred %q", b)
	}
}

func TestPeerPromoteSecretsFailRemovesNewPreferred(t *testing.T) {
	withTestRoot(t)
	idx := &PeersIndex{Version: 1, Peers: []ServicePeer{
		{ID: "p1", PublicIP: "203.0.113.2", State: "ready", Slot: SlotTable()[0]},
	}}
	_ = SavePeersIndex(idx)
	// no preferred files before promote
	writeFileFn = func(name string, data []byte, perm os.FileMode) error {
		if name == preferredIPSecretsPath {
			return fmt.Errorf("secrets denied")
		}
		return os.WriteFile(name, data, perm)
	}
	err := PeerPromote("p1")
	if err == nil {
		t.Fatal("expected fail")
	}
	if !strings.Contains(err.Error(), "secrets denied") {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(preferredIPPath); !os.IsNotExist(err) {
		b, _ := os.ReadFile(preferredIPPath)
		t.Fatalf("preferred should be removed, got %q", b)
	}
	idx2, _ := LoadPeersIndex()
	if idx2.Find("p1").State != "ready" {
		t.Fatalf("p1 should stay ready: %+v", idx2)
	}
}
