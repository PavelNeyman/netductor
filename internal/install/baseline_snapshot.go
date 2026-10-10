package install

import (
	"strings"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/integrity"
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
	if err := integrity.WriteManifest(); err != nil {
		fmt.Fprintf(os.Stderr, "integrity manifest: %v\n", err)
	}
	return nil
}


// BaselineDrift is a doctor-facing diff against the post-install snapshot.
type BaselineDrift struct {
	HaveSnapshot bool
	Created      string
	NewPorts     []string
	NewUnits     []string
}

// DiffHostBaseline compares the current host to the snapshot. Missing snapshot is not a failure.
func DiffHostBaseline() BaselineDrift {
	d := BaselineDrift{}
	dir := BaselineDir()
	if b, err := os.ReadFile(filepath.Join(dir, "created")); err == nil {
		d.HaveSnapshot = true
		d.Created = strings.TrimSpace(string(b))
	}
	if !d.HaveSnapshot {
		return d
	}
	d.NewPorts = newLines(filepath.Join(dir, "ss.txt"), capture("ss", "-tulnp"), portToken)
	d.NewUnits = newLines(filepath.Join(dir, "enabled-units.txt"), capture("systemctl", "list-unit-files", "--state=enabled"), unitToken)
	return d
}

func capture(cmd string, args ...string) string {
	out, _ := exec.Command(cmd, args...).CombinedOutput()
	return string(out)
}

func portToken(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 5 || !strings.Contains(line, "LISTEN") {
		return ""
	}
	return fields[4]
}

func unitToken(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.HasSuffix(fields[0], ".service") && !strings.HasSuffix(fields[0], ".timer") {
		return ""
	}
	if fields[1] != "enabled" {
		return ""
	}
	return fields[0]
}

func newLines(snapPath, now string, token func(string) string) []string {
	old := map[string]struct{}{}
	if b, err := os.ReadFile(snapPath); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if t := token(line); t != "" {
				old[t] = struct{}{}
			}
		}
	}
	var extra []string
	seen := map[string]struct{}{}
	for _, line := range strings.Split(now, "\n") {
		t := token(line)
		if t == "" {
			continue
		}
		if _, ok := old[t]; ok {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		extra = append(extra, t)
	}
	return extra
}
