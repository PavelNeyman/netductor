package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecondaryDirPrefersSecondary(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	root := StateDir()
	_ = os.MkdirAll(filepath.Join(root, "secondary"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "relay"), 0o755)
	if SecondaryDir() != filepath.Join(root, "secondary") {
		t.Fatal(SecondaryDir())
	}
}

func TestSecondaryDirLegacyRelay(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	root := StateDir()
	_ = os.MkdirAll(filepath.Join(root, "relay"), 0o755)
	if SecondaryDir() != filepath.Join(root, "relay") {
		t.Fatal(SecondaryDir())
	}
}
