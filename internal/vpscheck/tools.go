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
	optional := []string{"dnsutils", "fio"} // dig, fio
	var missing []string
	for _, b := range need {
		if _, err := exec.LookPath(b); err != nil {
			missing = append(missing, b)
		}
	}
	// also want dig
	if _, err := exec.LookPath("dig"); err != nil {
		missing = append(missing, "dnsutils")
	}
	if len(missing) == 0 {
		// still try optional fio quietly
		_ = optional
		if _, err := exec.LookPath("fio"); err != nil {
			if _, e2 := exec.LookPath("apt-get"); e2 == nil {
				cmd := exec.Command("apt-get", "install", "-y", "-qq", "fio")
				cmd.Env = append(cmd.Environ(), "DEBIAN_FRONTEND=noninteractive")
				_ = cmd.Run()
			}
		}
		return "tools ok", nil
	}
	pkgs := map[string]string{
		"curl": "curl", "iperf3": "iperf3", "sysbench": "sysbench",
		"dnsutils": "dnsutils", "dig": "dnsutils", "fio": "fio",
	}
	seen := map[string]bool{}
	var install []string
	for _, m := range missing {
		p := pkgs[m]
		if p == "" {
			p = m
		}
		if !seen[p] {
			seen[p] = true
			install = append(install, p)
		}
	}
	if _, err := exec.LookPath("apt-get"); err != nil {
		return "", fmt.Errorf("need apt-get to install %v (missing: %v)", install, missing)
	}
	cmd := exec.Command("apt-get", append([]string{"install", "-y", "-qq"}, install...)...)
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
	_ = timeout
	return strings.TrimSpace(string(out)), err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
