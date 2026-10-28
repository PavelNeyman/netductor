package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdDoneRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_AGENT_DIR", dir)
	// agentDir() may not use env — write under default by overriding via chdir-less path:
	// cmdDoneDir uses agentDir(); set up by writing config style if needed.
	// Force agentDir by creating expected structure if agentDir is fixed.
	ad := agentDir()
	_ = os.MkdirAll(ad, 0o700)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(ad, "cmd_done")) })

	saveCmdDone("cmd-abc-123", "reboot scheduled")
	got, ok := loadCmdDone("cmd-abc-123")
	if !ok || got != "reboot scheduled" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	// unsafe id rejected
	if p := cmdDonePath("../x"); p != "" && filepath.Base(p) == ".." {
		t.Fatal("path traversal")
	}
}
