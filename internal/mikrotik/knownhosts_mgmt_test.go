package mikrotik

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestForgetKnownHost(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_STATE", dir)
	// paths may use different env - write directly
	p := filepath.Join(paths.StateDir(), "mikrotik")
	_ = os.MkdirAll(p, 0o700)
	// seed via saveKH path - use Forget on empty
	err := ForgetKnownHost("no-such-host")
	if err == nil {
		t.Fatal("expected error")
	}
}
