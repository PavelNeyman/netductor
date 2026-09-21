package ci

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectImage(t *testing.T) {
	d := t.TempDir()
	if DetectImage(d) == "" {
		t.Fatal("generic")
	}
	_ = os.WriteFile(filepath.Join(d, "go.mod"), []byte("module x\n"), 0o644)
	if DetectImage(d) != ImageForGo() {
		t.Fatalf("got %s", DetectImage(d))
	}
}

func TestIsolationDefault(t *testing.T) {
	_ = os.Unsetenv("NETDUCTOR_CI_HOST")
	if !IsolationEnabled() {
		t.Fatal("expected isolation on")
	}
}
