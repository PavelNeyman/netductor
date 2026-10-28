package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdDoneRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_AGENT_DIR", dir)
	ad := agentDir()
	_ = os.MkdirAll(ad, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(ad, "cmd_done")) })

	if err := markCmdStarted("cmd-abc-123"); err != nil {
		t.Fatal(err)
	}
	if !cmdStarted("cmd-abc-123") {
		t.Fatal("expected started")
	}
	if err := saveCmdDone("cmd-abc-123", "reboot scheduled"); err != nil {
		t.Fatal(err)
	}
	got, ok := loadCmdDone("cmd-abc-123")
	if !ok || got != "reboot scheduled" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if cmdStarted("cmd-abc-123") {
		t.Fatal("completed should clear started semantics")
	}
}
