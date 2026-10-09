package incident

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Collect gathers journals/doctor/channel into a tar.gz under state/incidents.
func Collect(hours int) (dir, archive string, err error) {
	if hours < 1 {
		hours = 1
	}
	if hours > 24 {
		hours = 24
	}
	outDir := filepath.Join(paths.StateDir(), "incidents", time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", "", err
	}
	since := fmt.Sprintf("%d hours ago", hours)
	bin := "/usr/local/bin/netductor"
	if b, e := os.Executable(); e == nil && b != "" {
		bin = b
	}
	_ = exec.Command("bash", "-c",
		"journalctl -u sing-box -u netductor-api -u netductor-telegram-bot --since '"+since+"' --no-pager > '"+outDir+"/journal.txt' 2>&1; "+
			bin+" channel status > '"+outDir+"/channel.txt' 2>&1; "+
			bin+" doctor > '"+outDir+"/doctor.txt' 2>&1; "+
			"ss -tulnp > '"+outDir+"/ss.txt' 2>&1; "+
			bin+" secondary status > '"+outDir+"/secondary.txt' 2>&1 || true").Run()
	tar := outDir + ".tar.gz"
	if err := exec.Command("tar", "-C", filepath.Dir(outDir), "-czf", tar, filepath.Base(outDir)).Run(); err != nil {
		return outDir, "", fmt.Errorf("tar: %w", err)
	}
	return outDir, tar, nil
}
