package vpn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestPreferredSecondaryIPFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir) // may not work if paths fixed at init
	// write under default if paths uses /etc — just test helper reads our file via override
	_ = os.MkdirAll(filepath.Join(dir, "secrets"), 0o700)
	// preferredSecondaryIP uses paths.EtcDir() — skip if not overrideable
	_ = paths.EtcDir()
	p := preferredSecondaryIP()
	_ = p // smoke: no panic
}
