package vpscheck

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// EnsureTools installs packages needed for local checks (Debian/Ubuntu). Best-effort.
func EnsureTools() (string, error) {
	need := []string{"curl", "iperf3", "sysbench"}
	var missing []string
	for _, b := range need {
		if _, err := exec.LookPath(b); err != nil {
			missing = append(missing, b)
		}
	}
	if len(missing) == 0 {
		return "tools ok", nil
	}
	// map binary → package
	pkgs := map[string]string{"curl": "curl", "iperf3": "iperf3", "sysbench": "sysbench"}
	var install []string
	for _, m := range missing {
		if p := pkgs[m]; p != "" {
			install = append(install, p)
		}
	}
	if len(install) == 0 {
		return "", fmt.Errorf("missing tools: %v", missing)
	}
	if _, err := exec.LookPath("apt-get"); err != nil {
		return "", fmt.Errorf("need apt-get to install %v (missing binaries: %v)", install, missing)
	}
	cmd := exec.Command("apt-get", append([]string{"install", "-y"}, install...)...)
	cmd.Env = append(cmd.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("apt-get install %v: %w (%s)", install, err, truncate(string(out), 400))
	}
	return "installed " + strings.Join(install, ","), nil
}

func runCmd(timeout time.Duration, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	_ = timeout // reserved for CommandContext callers
	return strings.TrimSpace(string(out)), err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
