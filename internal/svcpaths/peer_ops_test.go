package svcpaths

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestPeerRemoveStopsBeforeDelete(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{Peers: []ServicePeer{{
		ID: "peer-b", State: "prepared",
		Slot: SlotTable()[1],
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	deleted := false
	oldStop, oldReload := peerRemoveStop, peerRemoveReload
	t.Cleanup(func() { peerRemoveStop, peerRemoveReload = oldStop, oldReload })
	peerRemoveStop = func(unit string) error { return fmt.Errorf("still active") }
	peerRemoveReload = func() error { deleted = true; return nil }
	err := PeerRemove("peer-b")
	if err == nil || !strings.Contains(err.Error(), "stop failed") {
		t.Fatalf("got %v", err)
	}
	if deleted {
		t.Fatal("reload ran after stop failure")
	}
	got, err := LoadPeersIndex()
	if err != nil {
		t.Fatal(err)
	}
	if got.Find("peer-b").State == "removed" {
		t.Fatal("peer marked removed after stop failure")
	}
}

func TestStopUnitIdempotentMissing(t *testing.T) {
	for _, msg := range []string{"Unit foo.service not found.", "No such file or directory", "not loaded"} {
		if !unitMissing(msg) {
			t.Fatalf("should treat as missing: %s", msg)
		}
	}
}
