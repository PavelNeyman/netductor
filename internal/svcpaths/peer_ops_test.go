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
	for _, msg := range []string{"Unit foo.service not found.", "unit foo.service not loaded", "unit foo does not exist"} {
		if !unitMissing(msg) {
			t.Fatalf("should treat as missing: %s", msg)
		}
	}
	for _, msg := range []string{"No such file or directory", "not loaded", "not-found"} {
		if unitMissing(msg) {
			t.Fatalf("bare %q is not a missing unit", msg)
		}
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
	removedFile := false
	oldStop, oldReload, oldSys := peerRemoveStop, peerRemoveReload, peerRemoveSystemctl
	oldRm := peerRemoveRemoveFile
	t.Cleanup(func() { peerRemoveRemoveFile = oldRm })
	peerRemoveRemoveFile = func(path string) error { removedFile = true; return nil }
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
	if removedFile {
		t.Fatal("config remove ran after bus error")
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

func TestPeerAddRejectsRemoved(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{Peers: []ServicePeer{{
		ID: "peer-old", State: "removed", PublicIP: "203.0.113.9",
		Slot: SlotTable()[1],
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	_, _, err := PeerAdd(PeerAddOpts{PeerID: "peer-old", PublicIP: "203.0.113.9", PrimaryIP: "198.51.100.1"})
	if err == nil || !strings.Contains(err.Error(), "removed") {
		t.Fatalf("add after remove must fail, got %v", err)
	}
}
