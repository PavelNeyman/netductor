package vpscheck

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// EnsureTools installs packages needed for local checks (Debian/Ubuntu). Best-effort.
func EnsureTools() (string, error) {
	return EnsureToolsCtx(context.Background())
}

// EnsureToolsCtx is cancelable (job cancel / timeout).
func EnsureToolsCtx(ctx context.Context) (string, error) {
	need := []string{"curl", "iperf3", "sysbench"}
	var missing []string
	for _, b := range need {
		if _, err := exec.LookPath(b); err != nil {
			missing = append(missing, b)
		}
	}
	if _, err := exec.LookPath("dig"); err != nil {
		missing = append(missing, "dnsutils")
	}
	if len(missing) == 0 {
		if _, err := exec.LookPath("fio"); err != nil {
			if _, e2 := exec.LookPath("apt-get"); e2 == nil {
				cmd := exec.CommandContext(ctx, "apt-get", "install", "-y", "-qq", "fio")
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
	cmd := exec.CommandContext(ctx, "apt-get", append([]string{"install", "-y", "-qq"}, install...)...)
	cmd.Env = append(cmd.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("apt-get install %v: %w (%s)", install, err, truncate(string(out), 400))
	}
	return "installed " + strings.Join(install, ","), nil
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// keep unused import happy if time needed later
var _ = time.Second
