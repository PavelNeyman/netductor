package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestValidatePeerID(t *testing.T) {
	if err := validatePeerID("peer-b"); err != nil {
		t.Fatal(err)
	}
	if err := validatePeerID("../evil"); err == nil {
		t.Fatal("expected invalid")
	}
	if err := validatePublicIP("8.8.8.1"); err != nil {
		t.Fatal(err)
	}
	for _, ip := range []string{"not-an-ip", "10.0.0.1", "100.64.0.1", "203.0.113.1", "198.51.100.1", "192.0.2.1", "::1"} {
		if err := validatePublicIP(ip); err == nil {
			t.Fatalf("expected non-public IPv4 %q to be rejected", ip)
		}
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
		ID: "peer-old", State: "removed", PublicIP: "8.8.8.9",
		Slot: SlotTable()[1],
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	_, _, err := PeerAdd(PeerAddOpts{PeerID: "peer-old", PublicIP: "8.8.8.9", PrimaryIP: "1.1.1.1"})
	if err == nil || !strings.Contains(err.Error(), "removed") {
		t.Fatalf("add after remove must fail, got %v", err)
	}
}

func TestPeerValidateSecondaryUsesLocalMaterial(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	old := peerValidateRole
	peerValidateRole = func() string { return "secondary" }
	t.Cleanup(func() { peerValidateRole = old })
	m := &Material{PeerID: "peer-b", Slot: 1, Secondary: "8.8.8.9", Primary: "1.1.1.1"}
	applySlotToMaterial(m, SlotTable()[1])
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := PeerValidateLocal("peer-b")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Errors {
		if strings.Contains(e, "peer material") {
			t.Fatalf("secondary should not require peers/<id>: %s", e)
		}
	}
}

func TestPeerRemoveDeletesMaterialAndUFW(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	slot := SlotTable()[1]
	idx := &PeersIndex{Peers: []ServicePeer{{
		ID: "peer-a", State: "prepared", PublicIP: "8.8.8.8", Slot: slot,
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	matDir := filepath.Join(dir, "peers", "peer-a")
	if err := os.MkdirAll(matDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(matDir, "material.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	ufwCalled := false
	oldStop, oldReload, oldUFW := peerRemoveStop, peerRemoveReload, peerRemoveUFW
	t.Cleanup(func() { peerRemoveStop, peerRemoveReload, peerRemoveUFW = oldStop, oldReload, oldUFW })
	peerRemoveStop = func(string) error { return nil }
	peerRemoveReload = func() error { return nil }
	peerRemoveUFW = func(publicIP, addrSSP, addrSPS string, wss int) error {
		ufwCalled = true
		if publicIP != "8.8.8.8" || addrSSP == "" {
			t.Fatalf("ufw args %s %s", publicIP, addrSSP)
		}
		return nil
	}
	if err := PeerRemove("peer-a"); err != nil {
		t.Fatal(err)
	}
	if !ufwCalled {
		t.Fatal("ufw cleanup not called")
	}
	if _, err := os.Stat(matDir); !os.IsNotExist(err) {
		t.Fatal("material dir still present")
	}
}

func TestPeerValidateSecondaryRequiresPeerID(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	old := peerValidateRole
	peerValidateRole = func() string { return "secondary" }
	t.Cleanup(func() { peerValidateRole = old })
	m := &Material{PeerID: "", Slot: 1, Secondary: "8.8.8.9", Primary: "1.1.1.1"}
	applySlotToMaterial(m, SlotTable()[1])
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "material.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := PeerValidateLocal("peer-b")
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("empty material PeerID must not match requested id")
	}
}

func TestPeerRemoveRetryAfterReloadFail(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	slot := SlotTable()[1]
	idx := &PeersIndex{Peers: []ServicePeer{{
		ID: "peer-r", State: "prepared", PublicIP: "8.8.8.7", Slot: slot,
	}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	reloads := 0
	oldStop, oldReload, oldUFW := peerRemoveStop, peerRemoveReload, peerRemoveUFW
	t.Cleanup(func() { peerRemoveStop, peerRemoveReload, peerRemoveUFW = oldStop, oldReload, oldUFW })
	peerRemoveStop = func(string) error { return nil }
	peerRemoveUFW = func(string, string, string, int) error { return nil }
	peerRemoveReload = func() error {
		reloads++
		if reloads == 1 {
			return fmt.Errorf("reload failed")
		}
		return nil
	}
	if err := PeerRemove("peer-r"); err == nil {
		t.Fatal("first remove should fail on reload")
	}
	got, err := LoadPeersIndex()
	if err != nil {
		t.Fatal(err)
	}
	if got.findAny("peer-r").State != "removing" {
		t.Fatalf("state after partial: %s", got.findAny("peer-r").State)
	}
	if err := PeerRemove("peer-r"); err != nil {
		t.Fatal(err)
	}
	got, _ = LoadPeersIndex()
	if got.findAny("peer-r").State != "removed" {
		t.Fatalf("retry did not finish: %s", got.findAny("peer-r").State)
	}
}

func TestPeerMarkReadyRejectsRemoved(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{Peers: []ServicePeer{{ID: "peer-x", State: "removed", Slot: SlotTable()[1]}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	if err := PeerMarkReady("peer-x", false); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Fatalf("ready on removed must fail, got %v", err)
	}
}

func TestPeerMarkReadyDoesNotReviveRemoved(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{Peers: []ServicePeer{{ID: "peer-y", State: "prepared", Slot: SlotTable()[1]}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	old := peerValidateForReady
	peerValidateForReady = func(id string) (*PeerValidateResult, error) {
		cur, err := LoadPeersIndex()
		if err != nil {
			return nil, err
		}
		if err := markPeerState(cur, id, "removed"); err != nil {
			return nil, err
		}
		return &PeerValidateResult{OK: true}, nil
	}
	t.Cleanup(func() { peerValidateForReady = old })
	if err := PeerMarkReady("peer-y", true); err == nil || !strings.Contains(err.Error(), "changed during validate") {
		t.Fatalf("ready must not revive removed, got %v", err)
	}
	got, _ := LoadPeersIndex()
	if got.findAny("peer-y").State != "removed" {
		t.Fatal("state changed")
	}
}

func TestRemovePeerUFWConfirmsGone(t *testing.T) {
	rules := "Status: active\n8789/tcp ALLOW 10.87.20.2\n8444/tcp ALLOW 8.8.8.9\n"
	old := peerUFWExec
	t.Cleanup(func() { peerUFWExec = old })
	peerUFWExec = func(args ...string) ([]byte, error) {
		if args[0] == "status" {
			return []byte(rules), nil
		}
		if args[0] == "delete" {
			src := args[3]
			rules = strings.ReplaceAll(rules, src, "")
			return []byte("deleted"), nil
		}
		return nil, fmt.Errorf("unexpected %v", args)
	}
	if err := removePeerUFW("8.8.8.9", "10.87.20.2", "", 8444); err != nil {
		t.Fatal(err)
	}
	peerUFWExec = func(args ...string) ([]byte, error) {
		if args[0] == "status" {
			return []byte("Status: active\n8789/tcp ALLOW 10.87.20.2\n"), nil
		}
		return []byte("deleted"), nil
	}
	if err := removePeerUFW("8.8.8.9", "10.87.20.2", "", 0); err == nil {
		t.Fatal("remaining rule must fail")
	}
	peerUFWExec = func(args ...string) ([]byte, error) {
		return []byte("fail"), fmt.Errorf("status failed")
	}
	if err := removePeerUFW("8.8.8.9", "10.87.20.2", "", 0); err == nil {
		t.Fatal("status error must fail")
	}
}

func TestPeerMarkReadyDoesNotDemoteActive(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	idx := &PeersIndex{ActiveID: "", Peers: []ServicePeer{{ID: "peer-z", State: "ready", Slot: SlotTable()[1]}}}
	if err := SavePeersIndex(idx); err != nil {
		t.Fatal(err)
	}
	old := peerValidateForReady
	peerValidateForReady = func(id string) (*PeerValidateResult, error) {
		cur, err := LoadPeersIndex()
		if err != nil {
			return nil, err
		}
		cur.ActiveID = id
		if err := markPeerState(cur, id, "active"); err != nil {
			return nil, err
		}
		return &PeerValidateResult{OK: true}, nil
	}
	t.Cleanup(func() { peerValidateForReady = old })
	if err := PeerMarkReady("peer-z", true); err == nil || !strings.Contains(err.Error(), "changed during validate") {
		t.Fatalf("ready must not overwrite active, got %v", err)
	}
	got, _ := LoadPeersIndex()
	if got.findAny("peer-z").State != "active" || got.ActiveID != "peer-z" {
		t.Fatalf("active lost: %+v", got)
	}
}

func TestRemovePeerUFWIgnoresOtherRule(t *testing.T) {
	old := peerUFWExec
	t.Cleanup(func() { peerUFWExec = old })
	peerUFWExec = func(args ...string) ([]byte, error) {
		if args[0] == "status" {
			return []byte("Status: active\n22/tcp ALLOW 8.8.8.9\n"), nil
		}
		return []byte("deleted"), nil
	}
	if err := removePeerUFW("8.8.8.9", "", "", 8444); err != nil {
		t.Fatalf("unrelated rule must not block remove: %v", err)
	}
}

func TestUFWRulePresentExact(t *testing.T) {
	status := "Status: active\n18444/tcp ALLOW 8.8.8.90\n22/tcp ALLOW 8.8.8.9\n"
	if ufwRulePresent(status, "8.8.8.9", "8444") {
		t.Fatal("prefix IP or leading port digit must not match")
	}
	if !ufwRulePresent(status, "8.8.8.90", "18444") {
		t.Fatal("exact longer IP and port should match")
	}
	if !ufwRulePresent("8444/tcp ALLOW 8.8.8.9\n", "8.8.8.9", "8444") {
		t.Fatal("exact rule should match")
	}
}

func TestPeerAddRefusesIncompleteOp(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	if err := os.MkdirAll(filepath.Join(dir, "peers", "peer-b"), 0o700); err != nil {
		t.Fatal(err)
	}
	op := `{"op":"add","peer_id":"peer-b","public_ip":"8.8.8.9","primary_ip":"1.1.1.1","slot_index":1,"phase":"material"}`
	if err := os.WriteFile(filepath.Join(dir, "peers", "peer-b", "op.json"), []byte(op), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := PeerAdd(PeerAddOpts{PeerID: "peer-b", PublicIP: "8.8.8.9", PrimaryIP: "1.1.1.1"})
	if err == nil || !strings.Contains(err.Error(), "peer recover") {
		t.Fatalf("incomplete op must refuse add, got %v", err)
	}
}

func TestPeersIndexRejectsUnknownSchemaButAcceptsLegacyUnversioned(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	path := peersIndexPath()

	if err := os.WriteFile(path, []byte(`{"peers":[]}`), 0o600); err != nil { t.Fatal(err) }
	legacy, err := LoadPeersIndex()
	if err != nil { t.Fatalf("legacy unversioned registry rejected: %v", err) }
	if legacy.Version != 1 { t.Fatalf("legacy version normalized to %d, want 1", legacy.Version) }

	if err := os.WriteFile(path, []byte(`{"version":2,"peers":[]}`), 0o600); err != nil { t.Fatal(err) }
	if _, err := LoadPeersIndex(); err == nil || !strings.Contains(err.Error(), "schema version 2 is unsupported") {
		t.Fatalf("future schema should fail closed, got %v", err)
	}
	if err := SavePeersIndex(&PeersIndex{Version: 2}); err == nil {
		t.Fatal("SavePeersIndex accepted unsupported schema")
	}
}


func TestPeerRegistryLockTimeout(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })

	lockPath := filepath.Join(dir, "peers.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN)

	oldTimeout := peerRegistryLockTimeout
	peerRegistryLockTimeout = 60 * time.Millisecond
	t.Cleanup(func() { peerRegistryLockTimeout = oldTimeout })

	started := time.Now()
	unlock, err := acquirePeerRegistryLock()
	if err == nil {
		unlock()
		t.Fatal("expected bounded lock acquisition to time out")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected lock error: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("lock timeout was not bounded: %s", elapsed)
	}
}


func TestPeerRecoverRegeneratesOnlyMissingAllocatingMaterial(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	if err := SavePeersIndex(&PeersIndex{}); err != nil {
		t.Fatal(err)
	}
	op := peerOp{Op: "add", PeerID: "peer-recover", PublicIP: "8.8.8.50", PrimaryIP: "1.1.1.10", SlotIndex: 1, Phase: "allocating"}
	if err := writePeerOpDefault(op); err != nil {
		t.Fatal(err)
	}

	oldGenerate, oldBootstrap := generateMaterialFn, bootstrapPrimaryFn
	oldWriteOp := writePeerOpFn
	t.Cleanup(func() {
		generateMaterialFn, bootstrapPrimaryFn, writePeerOpFn = oldGenerate, oldBootstrap, oldWriteOp
	})
	generated := 0
	bootstrapped := 0
	generateMaterialFn = func(primary, secondary string) (*Material, error) {
		generated++
		return &Material{SPPrivP: "private-primary", SPPrivS: "private-secondary"}, nil
	}
	bootstrapPrimaryFn = func(m *Material, _ BootstrapPrimaryOpts) error {
		bootstrapped++
		if m.PeerID != op.PeerID || m.Slot != op.SlotIndex || m.Secondary != op.PublicIP {
			t.Fatalf("recovered material identity mismatch: %+v", m)
		}
		return nil
	}
	writePeerOpFn = writePeerOpDefault

	status, err := RecoverIncompletePeerAdd(op.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if status != "recovered to prepared" {
		t.Fatalf("unexpected status %q", status)
	}
	if generated != 1 || bootstrapped != 1 {
		t.Fatalf("generator/bootstrap calls = %d/%d, want 1/1", generated, bootstrapped)
	}
	m, err := loadPeerMaterial(op.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := assertPeerMaterialIdentity(m, op.PeerID, op.PublicIP, op.SlotIndex); err != nil {
		t.Fatal(err)
	}
	idx, err := LoadPeersIndex()
	if err != nil {
		t.Fatal(err)
	}
	peer := idx.Find(op.PeerID)
	if peer == nil || peer.State != "prepared" || peer.Slot.Index != op.SlotIndex {
		t.Fatalf("recovered peer registry entry invalid: %+v", peer)
	}
	if _, err := os.Stat(peerOpPath(op.PeerID)); !os.IsNotExist(err) {
		t.Fatalf("operation journal should be cleared, stat err=%v", err)
	}
}

func TestPeerRecoverDoesNotRegenerateExistingCorruptMaterial(t *testing.T) {
	dir := t.TempDir()
	SetKeysRootForTest(dir)
	t.Cleanup(func() { SetKeysRootForTest("") })
	if err := SavePeersIndex(&PeersIndex{}); err != nil {
		t.Fatal(err)
	}
	op := peerOp{Op: "add", PeerID: "peer-corrupt", PublicIP: "8.8.8.51", PrimaryIP: "1.1.1.10", SlotIndex: 1, Phase: "allocating"}
	if err := writePeerOpDefault(op); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(peerMaterialPath(op.PeerID), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldGenerate := generateMaterialFn
	t.Cleanup(func() { generateMaterialFn = oldGenerate })
	generated := 0
	generateMaterialFn = func(string, string) (*Material, error) {
		generated++
		return &Material{SPPrivP: "new-p", SPPrivS: "new-s"}, nil
	}
	if _, err := RecoverIncompletePeerAdd(op.PeerID); err == nil || !strings.Contains(err.Error(), "refusing key regeneration") {
		t.Fatalf("expected fail-closed corrupt material error, got %v", err)
	}
	if generated != 0 {
		t.Fatalf("generator called %d times for existing corrupt material", generated)
	}
	b, err := os.ReadFile(peerMaterialPath(op.PeerID))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{broken" {
		t.Fatal("corrupt material was modified")
	}
}

func TestRecoverIncompletePeerAddRejectsRemovedIdentity(t *testing.T) {
	withTestRoot(t)
	if err := SavePeersIndex(&PeersIndex{Version: 1, Peers: []ServicePeer{{
		ID: "peer-retired", PublicIP: "8.8.8.40", State: "removed", Slot: SlotTable()[1],
	}}}); err != nil { t.Fatal(err) }
	if err := writePeerOp(peerOp{Op: "add", PeerID: "peer-retired", PublicIP: "8.8.8.40", PrimaryIP: "1.1.1.1", SlotIndex: 1, Phase: "bootstrap"}); err != nil { t.Fatal(err) }
	if _, err := RecoverIncompletePeerAdd("peer-retired"); err == nil || !strings.Contains(err.Error(), "refusing stale add recovery") {
		t.Fatalf("stale recovery must not resurrect removed identity: %v", err)
	}
}

 
func TestRecoverIncompletePeerAddRejectsDuplicateActiveIdentity(t *testing.T) {
	withTestRoot(t)
	peer := ServicePeer{ID: "peer-duplicate", PublicIP: "8.8.8.44", State: "prepared", Slot: SlotTable()[1]}
	other := peer
	other.PublicIP = "8.8.8.45"
	other.Slot = SlotTable()[2]
	if err := SavePeersIndex(&PeersIndex{Version: 1, Peers: []ServicePeer{peer, other}}); err != nil { t.Fatal(err) }
	if err := writePeerOp(peerOp{Op: "add", PeerID: peer.ID, PublicIP: peer.PublicIP, PrimaryIP: "1.1.1.1", SlotIndex: 1, Phase: "bootstrap"}); err != nil { t.Fatal(err) }
	if _, err := RecoverIncompletePeerAdd(peer.ID); err == nil || !strings.Contains(err.Error(), "ambiguous stale add recovery") {
		t.Fatalf("duplicate active identity must be rejected: %v", err)
	}
}

func TestRecoverIncompletePeerAddRejectsReassignedSlotAndIP(t *testing.T) {
	for _, tc := range []struct{name, opIP, registryIP string; slot int; want string}{
		{name: "ip", opIP: "8.8.8.41", registryIP: "8.8.8.41", slot: 2, want: "public IP"},
		{name: "slot", opIP: "8.8.8.42", registryIP: "8.8.8.43", slot: 1, want: "slot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withTestRoot(t)
			if err := SavePeersIndex(&PeersIndex{Version: 1, Peers: []ServicePeer{{
				ID: "peer-current", PublicIP: tc.registryIP, State: "prepared", Slot: SlotTable()[tc.slot],
			}}}); err != nil { t.Fatal(err) }
			if err := writePeerOp(peerOp{Op: "add", PeerID: "peer-stale", PublicIP: tc.opIP, PrimaryIP: "1.1.1.1", SlotIndex: tc.slot, Phase: "bootstrap"}); err != nil { t.Fatal(err) }
			if _, err := RecoverIncompletePeerAdd("peer-stale"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("stale recovery must reject reassigned %s: %v", tc.name, err)
			}
		})
	}
}
