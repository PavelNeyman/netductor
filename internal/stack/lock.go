package stack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const applyLockFile = "/var/lib/netductor/stack/apply.lock"
const applyLockTTL = 8 * time.Minute
const applyUnit = "netductor-stack-apply.service"

// applyUnitRunning is true if oneshot/service is still active or activating.
func applyUnitRunning() bool {
	out, err := exec.Command("systemctl", "is-active", applyUnit).Output()
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(out))
	return s == "active" || s == "activating"
}

// ApplyInProgress reports an active (or stale-cleared) stack apply lock.
func ApplyInProgress() (busy bool, tag string, started time.Time) {
	b, err := os.ReadFile(applyLockFile)
	if err != nil {
		return false, "", time.Time{}
	}
	line := strings.TrimSpace(string(b))
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) < 1 || parts[0] == "" {
		_ = ClearApplyLock()
		return false, "", time.Time{}
	}
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		_ = ClearApplyLock()
		return false, "", time.Time{}
	}
	// Stale TTL
	if time.Since(ts) > applyLockTTL {
		_ = ClearApplyLock()
		return false, "", time.Time{}
	}
	// Lock file without a live unit → crashed apply; free the lock
	if time.Since(ts) > 45*time.Second && !applyUnitRunning() {
		// give apply a short window to start after systemd-run
		_ = ClearApplyLock()
		return false, "", time.Time{}
	}
	t := ""
	if len(parts) > 1 {
		t = parts[1]
	}
	return true, t, ts
}

// TryAcquireApplyLock returns false if already busy.
func TryAcquireApplyLock(tag string) bool {
	if busy, _, _ := ApplyInProgress(); busy {
		return false
	}
	_ = os.MkdirAll(filepath.Dir(applyLockFile), 0o755)
	payload := time.Now().UTC().Format(time.RFC3339) + "\t" + strings.TrimSpace(tag)
	f, err := os.OpenFile(applyLockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if busy, _, _ := ApplyInProgress(); busy {
			return false
		}
		_ = os.WriteFile(applyLockFile, []byte(payload+"\n"), 0o644)
		return true
	}
	_, _ = f.WriteString(payload + "\n")
	_ = f.Close()
	return true
}

func ClearApplyLock() error {
	return os.Remove(applyLockFile)
}

// ForceClearApply unlocks stuck apply (admin). Stops oneshot unit if any.
func ForceClearApply() error {
	_ = exec.Command("systemctl", "stop", applyUnit).Run()
	_ = exec.Command("systemctl", "reset-failed", applyUnit).Run()
	return ClearApplyLock()
}

func ApplyLockStatusLine(ru bool) string {
	busy, tag, started := ApplyInProgress()
	if !busy {
		return ""
	}
	age := time.Since(started).Round(time.Second)
	if ru {
		return fmt.Sprintf("⏳ Обновление уже идёт: <code>%s</code> (%s). Подождите или сбросьте lock.", tag, age)
	}
	return fmt.Sprintf("⏳ Update already in progress: <code>%s</code> (%s). Wait or clear lock.", tag, age)
}
