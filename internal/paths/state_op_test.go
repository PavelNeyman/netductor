package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateDirHonorsEnv(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	if !strings.HasPrefix(StateDir(), os.TempDir()) && StateDir() != os.Getenv("NETDUCTOR_STATE") {
		// env set above
	}
	if StateDir() != os.Getenv("NETDUCTOR_STATE") {
		t.Fatalf("got %q", StateDir())
	}
}

func TestSecondaryDirUnderState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	if SecondaryDir() != filepath.Join(dir, "secondary") {
		t.Fatal(SecondaryDir())
	}
}
