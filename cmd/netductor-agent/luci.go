package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const luciUntilFile = "luci_until"

func luciUntilPath() string {
	return filepath.Join(agentDir(), luciUntilFile)
}

// luciCmd handles agent actions: luci_enable|luci_disable|luci_extend|luci_status.
// arg for enable/extend: hours=1 (default 1).
func luciCmd(action, arg string) string {
	hours := parseLuciHours(arg, 1)
	switch action {
	case "luci_status":
		return luciStatus()
	case "luci_disable":
		return luciDisable()
	case "luci_enable":
		return luciEnable(hours)
	case "luci_extend":
		return luciExtend(hours)
	default:
		return "luci: unknown action"
	}
}

func parseLuciHours(arg string, def float64) float64 {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return def
	}
	for _, p := range strings.Fields(arg) {
		if strings.HasPrefix(p, "hours=") {
			if v, err := strconv.ParseFloat(strings.TrimPrefix(p, "hours="), 64); err == nil && v > 0 && v <= 168 {
				return v
			}
		}
		if v, err := strconv.ParseFloat(p, 64); err == nil && v > 0 && v <= 168 {
			return v
		}
	}
	return def
}

func luciStatus() string {
	until := readLuciUntil()
	running := luciRunning()
	var untilS string
	if until.IsZero() {
		untilS = "none"
	} else {
		untilS = until.UTC().Format(time.RFC3339)
		if time.Now().After(until) {
			untilS += " (expired)"
		}
	}
	return fmt.Sprintf("running=%v until=%s", running, untilS)
}

func luciRunning() bool {
	out, _ := exec.Command("/etc/init.d/uhttpd", "status").CombinedOutput()
	s := strings.ToLower(string(out))
	return strings.Contains(s, "running") || strings.Contains(s, "active")
}

func luciEnable(hours float64) string {
	_ = os.MkdirAll(agentDir(), 0o700)
	until := time.Now().Add(time.Duration(hours * float64(time.Hour)))
	_ = os.WriteFile(luciUntilPath(), []byte(strconv.FormatInt(until.Unix(), 10)+"\n"), 0o600)
	// Ensure service not permanently disabled in rc
	_ = exec.Command("/etc/init.d/uhttpd", "enable").Run()
	out, err := exec.Command("/etc/init.d/uhttpd", "start").CombinedOutput()
	if err != nil {
		// try restart
		out2, err2 := exec.Command("/etc/init.d/uhttpd", "restart").CombinedOutput()
		if err2 != nil {
			return fmt.Sprintf("luci enable failed: %v %s %s", err, truncate(string(out), 400), truncate(string(out2), 400))
		}
	}
	return fmt.Sprintf("luci enabled until %s (%gh)", until.UTC().Format(time.RFC3339), hours)
}

func luciDisable() string {
	_ = os.Remove(luciUntilPath())
	out, err := exec.Command("/etc/init.d/uhttpd", "stop").CombinedOutput()
	if err != nil {
		return fmt.Sprintf("luci stop: %v %s", err, truncate(string(out), 400))
	}
	return "luci disabled"
}

func luciExtend(hours float64) string {
	until := readLuciUntil()
	base := time.Now()
	if !until.IsZero() && until.After(base) {
		base = until
	}
	newUntil := base.Add(time.Duration(hours * float64(time.Hour)))
	_ = os.MkdirAll(agentDir(), 0o700)
	_ = os.WriteFile(luciUntilPath(), []byte(strconv.FormatInt(newUntil.Unix(), 10)+"\n"), 0o600)
	if !luciRunning() {
		_ = exec.Command("/etc/init.d/uhttpd", "start").Run()
	}
	return fmt.Sprintf("luci extended until %s (+%gh)", newUntil.UTC().Format(time.RFC3339), hours)
}

func readLuciUntil() time.Time {
	b, err := os.ReadFile(luciUntilPath())
	if err != nil {
		return time.Time{}
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

// luciTickAutoDisable stops LuCI when TTL expired (call from agent loop).
func luciTickAutoDisable() {
	until := readLuciUntil()
	if until.IsZero() {
		return
	}
	if time.Now().Before(until) {
		return
	}
	if luciRunning() {
		_ = exec.Command("/etc/init.d/uhttpd", "stop").Run()
		fmt.Fprintf(os.Stderr, "luci: auto-disabled (TTL expired %s)\n", until.UTC().Format(time.RFC3339))
	}
	_ = os.Remove(luciUntilPath())
}


