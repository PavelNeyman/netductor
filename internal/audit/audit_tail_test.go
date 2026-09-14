package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestTail(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_STATE", dir)
	_ = os.MkdirAll(paths.StateDir(), 0o700)
	Log("t", "act", "tgt", "d")
	Log("t", "act2", "tgt2", "d2")
	ev := Tail(10)
	if len(ev) < 2 {
		t.Fatalf("%d events", len(ev))
	}
	_ = filepath.Join(dir, "x")
}
