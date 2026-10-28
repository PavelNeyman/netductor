package edge

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCmdResultIdempotentAfterPartialCompleteFail(t *testing.T) {
	setup(t)
	did, cid := "dev-a", "cmd-1"
	wrap := map[string]any{
		"pending": []map[string]any{
			{"id": cid, "device_id": did, "cmd": "echo", "arg": "x"},
		},
	}
	b, _ := json.Marshal(wrap)
	if err := os.WriteFile(commandsPath(), b, 0o600); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"device_id": did, "cmd_id": cid, "ok": true, "out": "hi"}

	// Inject failure after result is on disk: first POST returns error, leaves cmd pending.
	completeCmdSaveHook = func(deviceID, cmdID string, _ any) error {
		return fmt.Errorf("inject: queue save failed")
	}
	t.Cleanup(func() { completeCmdSaveHook = nil })

	err := CmdResult(payload)
	if err == nil {
		t.Fatal("expected error from injected complete failure")
	}
	completeCmdSaveHook = nil

	raw, err := os.ReadFile(resultsPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), cid) {
		t.Fatalf("result should be on disk after partial fail:\n%s", raw)
	}

	// Retry: must not append a second result line; must complete the queue.
	if err := CmdResult(payload); err != nil {
		t.Fatalf("retry: %v", err)
	}
	raw2, _ := os.ReadFile(resultsPath())
	n := 0
	for _, line := range strings.Split(string(raw2), "\n") {
		if strings.Contains(line, cid) && strings.Contains(line, did) {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want 1 result line, got %d:\n%s", n, raw2)
	}
	pending, err := pendingCmdExistsUnlocked(did, cid)
	if err != nil {
		t.Fatal(err)
	}
	if pending {
		t.Fatal("command should be removed from queue after successful retry")
	}
}

func TestCmdResultSameCmdIDDifferentDevice(t *testing.T) {
	setup(t)
	cid := "shared-id"
	p1 := map[string]any{"device_id": "d1", "cmd_id": cid, "out": "a"}
	p2 := map[string]any{"device_id": "d2", "cmd_id": cid, "out": "b"}
	if err := CmdResult(p1); err != nil {
		t.Fatal(err)
	}
	if err := CmdResult(p2); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(resultsPath())
	var count int
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		count++
	}
	if count < 2 {
		t.Fatalf("want 2 device results, got:\n%s", raw)
	}
}

func TestPendingCmdExistsReadError(t *testing.T) {
	setup(t)
	// Point commands path at a directory so LoadJSON fails with a real error.
	p := commandsPath()
	_ = os.Remove(p)
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(p) })
	_, err := pendingCmdExistsUnlocked("d", "c")
	if err == nil {
		t.Fatal("expected error when commands path is a directory")
	}
}
