package fleet

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SyncPaths is deprecated. Secondary is VPN entry only; lampac/data mirror removed.
// Kept as no-op so old timers/CLI do not fail hard.
func SyncPaths(peerSSH string) error {
	fmt.Fprintln(os.Stderr, "fleet sync: no-op (secondary is VPN entry only; data mirror removed)")
	_ = os.MkdirAll(filepath.Join(paths.StateDir(), "fleet"), 0o700)
	line := time.Now().UTC().Format(time.RFC3339) + " no-op\n"
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "fleet", "last_sync"), []byte(line), 0o644)
	return nil
}
