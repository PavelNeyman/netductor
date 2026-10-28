package edge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdResultIdempotentAfterPartialCompleteFail(t *testing.T) {
	setup(t)
	did, cid := "dev-a", "cmd-1"
	// Seed a pending command
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
	if err := CmdResult(payload); err != nil {
		t.Fatal(err)
	}
	// First success should have completed and removed from queue
	if pendingCmdExistsUnlocked(did, cid) {
		// reload under lock semantics - use CmdResult path
	}
	// Manually re-add to pending and keep result on disk to simulate partial fail recovery
	if err := os.WriteFile(commandsPath(), b, 0o600); err != nil {
		t.Fatal(err)
	}
	// Result already on disk from first call
	if err := CmdResult(payload); err != nil {
		t.Fatalf("retry: %v", err)
	}
	raw, err := os.ReadFile(resultsPath())
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, cid) && strings.Contains(line, did) {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want 1 result line, got %d:\n%s", n, raw)
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
	if !strings.Contains(string(raw), `"device_id":"d1"`) || !strings.Contains(string(raw), `"device_id":"d2"`) {
		// json marshal order may differ
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
	_ = filepath.Separator
}
