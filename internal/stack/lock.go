package stack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const applyLockFile = "/var/lib/netductor/stack/apply.lock"
const applyLockTTL = 8 * time.Minute
const applyUnit = "netductor-stack-apply.service"

// applyUnitRunning is true if oneshot/service is still active or activating.
func applyUnitRunning() bool {
	out, err := exec.Command("systemctl", "is-active", applyUnit).Output()
	if err != nil { return false }
	s := strings.TrimSpace(string(out))
	return s == "active" || s == "activating"
}

func applyPIDRunning(pidText string) bool {
	pid, err := strconv.Atoi(strings.TrimSpace(pidText))
	if err != nil || pid <= 1 { return false }
	err = exec.Command("kill", "-0", strconv.Itoa(pid)).Run()
	return err == nil
}

// ApplyInProgress reports a live apply owner and clears only demonstrably stale locks.
func ApplyInProgress() (busy bool, tag string, started time.Time) {
	b, err := os.ReadFile(applyLockFile)
	if err != nil { return false, "", time.Time{} }
	parts := strings.SplitN(strings.TrimSpace(string(b)), "\t", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" {
		return false, "", time.Time{}
	}
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil { return false, "", time.Time{} }
	alive := applyPIDRunning(parts[2]) || applyUnitRunning()
	if !alive || time.Since(ts) > applyLockTTL {
		_ = ClearApplyLock()
		return false, "", time.Time{}
	}
	return true, parts[1], ts
}

// TryAcquireApplyLock creates one durable exclusive owner record. It never falls
// back to a non-exclusive write when O_EXCL reports an existing lock.
func TryAcquireApplyLock(tag string) bool {
	if busy, _, _ := ApplyInProgress(); busy { return false }
	if err := os.MkdirAll(filepath.Dir(applyLockFile), 0o755); err != nil { return false }
	payload := time.Now().UTC().Format(time.RFC3339) + "\t" + strings.TrimSpace(tag) + "\t" + strconv.Itoa(os.Getpid()) + "\n"
	f, err := os.OpenFile(applyLockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil { return false }
	if _, err := f.WriteString(payload); err != nil { _ = f.Close(); _ = os.Remove(applyLockFile); return false }
	if err := f.Sync(); err != nil { _ = f.Close(); _ = os.Remove(applyLockFile); return false }
	if err := f.Close(); err != nil { _ = os.Remove(applyLockFile); return false }
	if err := syncDir(filepath.Dir(applyLockFile)); err != nil { _ = os.Remove(applyLockFile); return false }
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
