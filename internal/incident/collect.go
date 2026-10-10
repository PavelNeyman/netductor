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
	runTo := func(path string, name string, args ...string) {
		f, err := os.Create(path)
		if err != nil {
			return
		}
		defer f.Close()
		cmd := exec.Command(name, args...)
		cmd.Stdout = f
		cmd.Stderr = f
		_ = cmd.Run()
	}
	runTo(filepath.Join(outDir, "journal.txt"), "journalctl",
		"-u", "sing-box", "-u", "netductor-api", "-u", "netductor-telegram-bot",
		"--since", since, "--no-pager")
	runTo(filepath.Join(outDir, "channel.txt"), bin, "channel", "status")
	runTo(filepath.Join(outDir, "doctor.txt"), bin, "doctor")
	runTo(filepath.Join(outDir, "ss.txt"), "ss", "-tulnp")
	runTo(filepath.Join(outDir, "secondary.txt"), bin, "secondary", "status")

	tar := outDir + ".tar.gz"
	if err := exec.Command("tar", "-C", filepath.Dir(outDir), "-czf", tar, filepath.Base(outDir)).Run(); err != nil {
		return outDir, "", fmt.Errorf("tar: %w", err)
	}
	return outDir, tar, nil
}
