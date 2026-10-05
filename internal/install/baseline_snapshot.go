package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// BaselineDir is where post-install host snapshots live (HOST-AUDIT §7).
func BaselineDir() string {
	return filepath.Join(paths.StateDir(), "baseline")
}

// SnapshotHostBaseline records ss, enabled units, dpkg, ufw/iptables for later doctor diff.
// Safe to call repeatedly; overwrites previous snapshot files.
func SnapshotHostBaseline() error {
	dir := BaselineDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	write := func(file string, cmd string, args ...string) {
		out, err := exec.Command(cmd, args...).CombinedOutput()
		body := out
		if err != nil && len(body) == 0 {
			body = []byte(err.Error() + "\n")
		}
		_ = os.WriteFile(filepath.Join(dir, file), body, 0o644)
	}
	write("ss.txt", "ss", "-tulnp")
	write("enabled-units.txt", "systemctl", "list-unit-files", "--state=enabled")
	write("dpkg.txt", "dpkg", "-l")
	if _, err := exec.LookPath("ufw"); err == nil {
		write("ufw.txt", "ufw", "status", "verbose")
	} else if _, err := exec.LookPath("iptables-save"); err == nil {
		write("iptables.txt", "iptables-save")
	}
	_ = os.WriteFile(filepath.Join(dir, "created"),
		[]byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644)
	fmt.Fprintln(os.Stderr, "host baseline snapshot →", dir)
	return nil
}
