package notify

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// MigrateAwayFromTopics removes on-disk private-topic state (one-shot safe).
// Call once at bot/API start. Channel alerts and admin DM are unaffected.
func MigrateAwayFromTopics() {
	dir := filepath.Join(paths.StateDir(), "tg")
	for _, name := range []string{
		"topics.json",
		"topics-backup.json",
		"topics_disabled",
	} {
		_ = os.Remove(filepath.Join(dir, name))
	}
	ClearHubMsg()
	marker := filepath.Join(dir, "topics_removed")
	if _, err := os.Stat(marker); err == nil {
		return
	}
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(marker, []byte("1\n"), 0o600)
	fmt.Fprintln(os.Stderr, "notify: removed TG topics state (channel + flat DM only)")
}
