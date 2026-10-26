package svcpaths

import (
	"fmt"
	"os/exec"
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
	for _, msg := range []string{"Unit foo.service not found.", "not loaded", "unit does not exist"} {
		if !unitMissing(msg) {
			t.Fatalf("should treat as missing: %s", msg)
		}
	}
	if unitMissing("No such file or directory") {
		t.Fatal("bare no such file is not a missing unit")
	}
}

func TestPeerRemoveSystemctlNotStarted(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{Peers: []ServicePeer{{
		ID: "peer-c", State: "prepared",
		Slot: SlotTable()[1],
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	reloaded := false
	oldStop, oldReload := peerRemoveStop, peerRemoveReload
	t.Cleanup(func() { peerRemoveStop, peerRemoveReload = oldStop, oldReload })
	peerRemoveStop = stopUnitIdempotent
	peerRemoveReload = func() error { reloaded = true; return nil }
	t.Setenv("PATH", t.TempDir())
	err := PeerRemove("peer-c")
	if err == nil || !strings.Contains(err.Error(), "not started") {
		t.Fatalf("got %v", err)
	}
	if reloaded {
		t.Fatal("reload ran when systemctl did not start")
	}
	got, err := LoadPeersIndex()
	if err != nil {
		t.Fatal(err)
	}
	if got.Find("peer-c").State == "removed" {
		t.Fatal("peer marked removed when systemctl did not start")
	}
}

func TestPeerRemoveBusErrorKeepsConfigs(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{Peers: []ServicePeer{{
		ID: "peer-d", State: "prepared",
		Slot: SlotTable()[1],
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	reloaded := false
	oldStop, oldReload, oldSys := peerRemoveStop, peerRemoveReload, peerRemoveSystemctl
	t.Cleanup(func() {
		peerRemoveStop, peerRemoveReload, peerRemoveSystemctl = oldStop, oldReload, oldSys
	})
	peerRemoveStop = stopUnitIdempotent
	peerRemoveReload = func() error { reloaded = true; return nil }
	peerRemoveSystemctl = func(args ...string) ([]byte, error) {
		cmd := exec.Command("false")
		err := cmd.Run()
		return []byte("Failed to connect to bus: No such file or directory"), err
	}
	err := PeerRemove("peer-d")
	if err == nil || strings.Contains(err.Error(), "not started") {
		t.Fatalf("bus error must be a stop failure, got %v", err)
	}
	if reloaded {
		t.Fatal("reload ran after bus error")
	}
	got, err := LoadPeersIndex()
	if err != nil {
		t.Fatal(err)
	}
	if got.Find("peer-d").State == "removed" {
		t.Fatal("peer marked removed after bus error")
	}
}

func TestUnitMissingRejectsBusError(t *testing.T) {
	if unitMissing("Failed to connect to bus: No such file or directory") {
		t.Fatal("bus error must not look like a missing unit")
	}
	if !unitMissing("Unit nd-wss-sp-server.service not found.") {
		t.Fatal("real missing unit should match")
	}
}
