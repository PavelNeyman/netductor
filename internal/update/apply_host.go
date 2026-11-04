package update

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HostApplyOpts controls a local host component update (tg/agent; node often uses stack.Apply).
type HostApplyOpts struct {
	Component      ComponentID
	InstanceID     InstanceID // default "local"
	Tag            string
	DestPath       string
	Unit           string // systemd unit to try-restart; empty = skip
	Restart        bool
	IdempotencyKey string
	JobID          string
	// RequireVersionProbe when true (default for Restart) checks binary "version" output contains target.
	SkipVersionProbe bool
	// SkipDownload is for tests.
	SkipDownload bool
	// AfterDownload optional hook (tests).
	AfterDownload func() error
}

// DefaultHostRegistry is process-local; CLI uses it to prevent concurrent host applies.
var DefaultHostRegistry = NewJobRegistry()

// ApplyHostComponent runs resolve → download to .new → keep .prev → atomic rename → optional restart → health.
// States follow the Phase 1 machine. Does not write VERSION for multi-binary stacks (caller may WriteVERSION).
// Cross-process flock serializes Apply/Rollback for the same component/instance.
func ApplyHostComponent(o HostApplyOpts) (*UpdateJob, error) {
	if o.InstanceID == "" {
		o.InstanceID = "local"
	}
	var out *UpdateJob
	var outErr error
	lockErr := withHostComponentLock(o.Component, o.InstanceID, func() error {
		out, outErr = applyHostComponentLocked(o)
		return outErr
	})
	if lockErr != nil && out == nil {
		return nil, lockErr
	}
	return out, outErr
}

func applyHostComponentLocked(o HostApplyOpts) (*UpdateJob, error) {
	if o.JobID == "" {
		o.JobID = fmt.Sprintf("%s-%s-%d", o.Component, o.InstanceID, time.Now().UnixNano())
	}
	if !ValidComponent(o.Component) {
		return nil, fmt.Errorf("update: unsupported component %q", o.Component)
	}
	tag, err := ValidReleaseTag(o.Tag)
	if err != nil {
		return nil, err
	}
	res, err := ResolveReleaseAsset(o.Component, PlatformFromRuntime(), tag)
	if err != nil {
		if !o.SkipDownload {
			return nil, err
		}
		res = ResolvedAsset{Component: o.Component, Tag: tag, Name: AssetName(o.Component, PlatformFromRuntime())}
	}
	job := &UpdateJob{
		JobID:          o.JobID,
		Component:      o.Component,
		InstanceID:     o.InstanceID,
		TargetVersion:  strings.TrimPrefix(tag, "v"),
		Asset:          res.Name,
		AssetSHA256:    res.SHA256,
		State:          StateQueued,
		IdempotencyKey: o.IdempotencyKey,
	}
	cur, err := DefaultHostRegistry.Begin(job)
	if err != nil {
		return cur, err
	}
	mark := func(to JobState, msg string) (*UpdateJob, error) {
		return DefaultHostRegistry.Transition(o.Component, o.InstanceID, to, msg)
	}
	if _, err := mark(StateDownloading, ""); err != nil {
		return cur, err
	}

	dest := o.DestPath
	newPath := dest + ".new"
	prevPath := dest + ".prev"

	if !o.SkipDownload {
		if dest == "" {
			j, _ := mark(StateFailed, "install: empty dest")
			return j, fmt.Errorf("update: empty dest path")
		}
		_ = os.Remove(newPath)
		if err := DownloadNamedAsset(tag, res.Name, newPath); err != nil {
			_ = os.Remove(newPath)
			j, _ := mark(StateFailed, err.Error())
			return j, err
		}
	}
	if o.AfterDownload != nil {
		if err := o.AfterDownload(); err != nil {
			j, _ := mark(StateFailed, err.Error())
			return j, err
		}
	}
	if _, err := mark(StateVerifying, ""); err != nil {
		return cur, err
	}
	if !o.SkipDownload {
		st, err := os.Stat(newPath)
		if err != nil || st.Size() == 0 {
			_ = os.Remove(newPath)
			j, _ := mark(StateFailed, "install: downloaded file missing or empty")
			return j, fmt.Errorf("update: downloaded file missing or empty")
		}
	}
	if _, err := mark(StateInstalling, ""); err != nil {
		return cur, err
	}
	if !o.SkipDownload {
		if err := atomicReplaceBinary(dest, newPath, prevPath); err != nil {
			_ = os.Remove(newPath)
			j, _ := mark(StateFailed, err.Error())
			return j, err
		}
	}

	failHealth := func(msg error) (*UpdateJob, error) {
		if !o.SkipDownload {
			if _, e := os.Stat(prevPath); e == nil {
				_, journalErr := mark(StateRollbackInProgress, "")
				// A journal write failure must not prevent restoring the known-good
				// executable; preserve the journal error and report rollback as
				// incomplete rather than leaving the failed binary installed.
				if rerr := restorePrevBinary(dest, prevPath); rerr != nil {
					msg = ErrRollbackFailed(msg, errors.Join(journalErr, rerr))
					j, _ := mark(StateFailed, msg.Error())
					return j, msg
				}
				if journalErr != nil {
					msg = ErrRollbackFailed(msg, fmt.Errorf("rollback journal transition: %w", journalErr))
					j, _ := mark(StateFailed, msg.Error())
					return j, msg
				}
				if o.Restart && o.Unit != "" {
					if rerr := exec.Command("systemctl", "restart", o.Unit).Run(); rerr != nil {
						msg = ErrRollbackFailed(msg, fmt.Errorf("restart restored %s: %w", o.Unit, rerr))
						j, _ := mark(StateFailed, msg.Error())
						return j, msg
					}
					if rerr := healthUnitActive(o.Unit); rerr != nil {
						msg = ErrRollbackFailed(msg, fmt.Errorf("restored unit health: %w", rerr))
						j, _ := mark(StateFailed, msg.Error())
						return j, msg
					}
					if rerr := probeRunningUnitBinary(o.Unit, dest); rerr != nil {
						msg = ErrRollbackFailed(msg, fmt.Errorf("restored process identity: %w", rerr))
						j, _ := mark(StateFailed, msg.Error())
						return j, msg
					}
				}
				j, _ := mark(StateRolledBack, msg.Error())
				return j, msg
			}
		}
		j, _ := mark(StateFailed, msg.Error())
		return j, msg
	}

	if o.Restart && o.Unit != "" {
		if _, err := mark(StateRestarting, ""); err != nil {
			return cur, err
		}
		if err := exec.Command("systemctl", "restart", o.Unit).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "update: systemctl restart %s: %v\n", o.Unit, err)
		}
		if _, err := mark(StateHealthChecking, ""); err != nil {
			return cur, err
		}
		if err := healthUnitActive(o.Unit); err != nil {
			return failHealth(err)
		}
		if !o.SkipVersionProbe {
			if err := probeRunningUnitVersion(o.Unit, job.TargetVersion); err != nil {
				return failHealth(err)
			}
		}
	} else {
		if _, err := mark(StateHealthChecking, ""); err != nil {
			return cur, err
		}
		if !o.SkipDownload && !o.SkipVersionProbe && dest != "" {
			// File replaced but not restarted: still try version string when binary can run.
			if err := probeBinaryVersion(dest, job.TargetVersion); err != nil {
				// Soft: offline tests / non-exec environments — only fail when file was replaced and probe is required.
				// Keep strict when Restart was requested; without restart, log-only soft fail is risky — fail closed if dest is executable path.
				if st, e := os.Stat(dest); e == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0 {
					return failHealth(err)
				}
			}
		}
	}
	j, err := mark(StateSucceeded, "")
	return j, err
}

// atomicReplaceBinary moves newPath into dest, keeping previous dest as prevPath when present.
func atomicReplaceBinary(dest, newPath, prevPath string) error {
	if strings.TrimSpace(dest) == "" || strings.TrimSpace(newPath) == "" || strings.TrimSpace(prevPath) == "" {
		return fmt.Errorf("update: empty binary path")
	}
	parent := filepath.Clean(filepath.Dir(dest))
	if filepath.Clean(filepath.Dir(newPath)) != parent || filepath.Clean(filepath.Dir(prevPath)) != parent {
		return fmt.Errorf("update: new and previous binaries must be staged beside destination")
	}
	if err := os.MkdirAll(parent, 0o755); err != nil { return fmt.Errorf("create destination directory: %w", err) }
	newInfo, err := os.Lstat(newPath)
	if err != nil { return fmt.Errorf("inspect staged binary: %w", err) }
	if !newInfo.Mode().IsRegular() || newInfo.Size() == 0 { return fmt.Errorf("update: staged binary is not a non-empty regular file") }
	if err := os.Chmod(newPath, 0o755); err != nil { return fmt.Errorf("chmod staged binary: %w", err) }
	if err := hostSyncFile(newPath); err != nil { return fmt.Errorf("sync staged binary: %w", err) }

	hadPrevious := false
	if st, err := os.Lstat(dest); err == nil {
		if !st.Mode().IsRegular() { return fmt.Errorf("update: destination is not a regular file") }
		if err := hostCopyFileAtomic(dest, prevPath, 0o755); err != nil { return fmt.Errorf("preserve previous binary: %w", err) }
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect destination binary: %w", err)
	}

	if err := os.Rename(newPath, dest); err != nil {
		return fmt.Errorf("atomic install binary: %w", err)
	}
	if err := hostSyncDir(parent); err != nil {
		if hadPrevious {
			if restoreErr := hostCopyFileAtomic(prevPath, dest, 0o755); restoreErr != nil {
				return fmt.Errorf("sync destination directory: %v; CRITICAL: previous binary restore failed: %w", err, restoreErr)
			}
			return fmt.Errorf("sync destination directory: %v; previous binary restored", err)
		}
		removeErr := os.Remove(dest)
		syncErr := hostSyncDir(parent)
		if removeErr != nil || syncErr != nil { return fmt.Errorf("sync destination directory: %v; cleanup new binary failed: remove=%v sync=%v", err, removeErr, syncErr) }
		return fmt.Errorf("sync destination directory: %v; newly installed binary removed", err)
	}
	return nil
}

// restorePrevBinary atomically restores the saved image without first removing the live executable.
func restorePrevBinary(dest, prevPath string) error {
	st, err := os.Lstat(prevPath)
	if err != nil { return err }
	if !st.Mode().IsRegular() || st.Size() == 0 { return fmt.Errorf("update: previous binary is not a non-empty regular file") }
	if filepath.Clean(filepath.Dir(dest)) != filepath.Clean(filepath.Dir(prevPath)) {
		return fmt.Errorf("update: previous binary must be beside destination")
	}
	return hostCopyFileAtomic(prevPath, dest, 0o755)
}

func hostSyncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil { return err }
	if err := f.Sync(); err != nil { _ = f.Close(); return err }
	return f.Close()
}

func hostSyncDir(path string) error {
	f, err := os.Open(path)
	if err != nil { return err }
	if err := f.Sync(); err != nil { _ = f.Close(); return err }
	return f.Close()
}

func hostCopyFileAtomic(src, dst string, mode os.FileMode) error {
	if filepath.Clean(filepath.Dir(src)) != filepath.Clean(filepath.Dir(dst)) {
		return fmt.Errorf("source and destination must share a directory")
	}
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".netductor-host-copy-*")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, in); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Chmod(mode); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Sync(); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	if err := os.Rename(tmpName, dst); err != nil { return err }
	return hostSyncDir(filepath.Dir(dst))
}
func healthUnitActive(unit string) error {
	out, err := exec.Command("systemctl", "is-active", unit).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil || (s != "active" && s != "activating") {
		return fmt.Errorf("health: unit %s is %q", unit, s)
	}
	return nil
}

// probeRunningUnitBinary confirms that systemd restarted the unit from the
// restored on-disk executable, rather than leaving the failed process in memory.
func probeRunningUnitBinary(unit, binaryPath string) error {
	out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("health: MainPID for %s: %v (%s)", unit, err, truncateOut(string(out), 80))
	}
	pid := strings.TrimSpace(string(out))
	if pid == "" || pid == "0" {
		return fmt.Errorf("health: unit %s has no MainPID", unit)
	}
	procExe := filepath.Join("/proc", pid, "exe")
	procInfo, err := os.Stat(procExe)
	if err != nil {
		return fmt.Errorf("health: inspect running executable for %s: %w", unit, err)
	}
	destInfo, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("health: inspect restored executable %s: %w", binaryPath, err)
	}
	if !os.SameFile(procInfo, destInfo) {
		return fmt.Errorf("health: unit %s is not running the restored executable %s", unit, binaryPath)
	}
	return nil
}

// probeRunningUnitVersion checks the systemd unit MainPID executable version string.
func probeRunningUnitVersion(unit, wantVer string) error {
	out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("health: MainPID for %s: %v (%s)", unit, err, truncateOut(string(out), 80))
	}
	pid := strings.TrimSpace(string(out))
	if pid == "" || pid == "0" {
		return fmt.Errorf("health: unit %s has no MainPID (not running?)", unit)
	}
	return probeBinaryVersion(filepath.Join("/proc", pid, "exe"), wantVer)
}

// probeBinaryVersion runs "<path> version" and requires output to contain the target version string.
func probeBinaryVersion(path, wantVer string) error {
	wantVer = strings.TrimPrefix(strings.TrimSpace(wantVer), "v")
	if path == "" || wantVer == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("health: binary missing: %w", err)
	}
	out, err := exec.Command(path, "version").CombinedOutput()
	s := string(out)
	if err != nil {
		// try --version
		out2, err2 := exec.Command(path, "--version").CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("health: version probe failed: %v (%s)", err, truncateOut(s, 200))
		}
		s = string(out2)
	}
	if !strings.Contains(s, wantVer) {
		return fmt.Errorf("health: running version does not contain %q (got %q)", wantVer, truncateOut(s, 120))
	}
	return nil
}

func truncateOut(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// RollbackHostComponent restores dest from dest+".prev" if present.
// Supported for host tg/agent (and any path using the same .prev layout).
// Does not stack-rollback node — use stack.Rollback for that.
func RollbackHostComponent(o HostApplyOpts) (*UpdateJob, error) {
	if o.InstanceID == "" {
		o.InstanceID = "local"
	}
	var out *UpdateJob
	var outErr error
	lockErr := withHostComponentLock(o.Component, o.InstanceID, func() error {
		out, outErr = rollbackHostComponentLocked(o)
		return outErr
	})
	if lockErr != nil && out == nil {
		return nil, lockErr
	}
	return out, outErr
}

func rollbackHostComponentLocked(o HostApplyOpts) (*UpdateJob, error) {
	if o.JobID == "" {
		o.JobID = fmt.Sprintf("rollback-%s-%s-%d", o.Component, o.InstanceID, time.Now().UnixNano())
	}
	if !ValidComponent(o.Component) {
		return nil, fmt.Errorf("update: unsupported component %q", o.Component)
	}
	if o.Component == ComponentNode {
		return nil, fmt.Errorf("update: node rollback via stack.rollback, not host component")
	}
	if o.DestPath == "" {
		return nil, fmt.Errorf("update: dest path required")
	}
	prevPath := o.DestPath + ".prev"
	if _, err := os.Stat(prevPath); err != nil {
		return nil, fmt.Errorf("update: no previous binary at %s", prevPath)
	}
	job := &UpdateJob{
		JobID:      o.JobID,
		Component:  o.Component,
		InstanceID: o.InstanceID,
		Asset:      filepath.Base(o.DestPath),
		State:      StateQueued,
	}
	cur, err := DefaultHostRegistry.Begin(job)
	if err != nil {
		return cur, err
	}
	mark := func(to JobState, msg string) (*UpdateJob, error) {
		return DefaultHostRegistry.Transition(o.Component, o.InstanceID, to, msg)
	}
	if _, err := mark(StateRollbackInProgress, ""); err != nil {
		return cur, err
	}
	if err := restorePrevBinary(o.DestPath, prevPath); err != nil {
		j, _ := mark(StateFailed, err.Error())
		return j, fmt.Errorf("rollback: %w", err)
	}
	if o.Restart && o.Unit != "" {
		if err := exec.Command("systemctl", "restart", o.Unit).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "update: systemctl restart %s after rollback: %v\n", o.Unit, err)
		}
		if err := healthUnitActive(o.Unit); err != nil {
			j, _ := mark(StateFailed, err.Error())
			return j, err
		}
	}
	j, err := mark(StateRolledBack, "restored from .prev")
	return j, err
}
