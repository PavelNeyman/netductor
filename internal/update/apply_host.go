package update

import (
	"fmt"
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
func ApplyHostComponent(o HostApplyOpts) (*UpdateJob, error) {
	if o.InstanceID == "" {
		o.InstanceID = "local"
	}
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
		// Best-effort restore previous binary if we kept one.
		if !o.SkipDownload {
			_ = restorePrevBinary(dest, prevPath)
		}
		j, _ := mark(StateFailed, msg.Error())
		return j, msg
	}

	if o.Restart && o.Unit != "" {
		if _, err := mark(StateRestarting, ""); err != nil {
			return cur, err
		}
		_ = exec.Command("systemctl", "try-restart", o.Unit).Run()
		if _, err := mark(StateHealthChecking, ""); err != nil {
			return cur, err
		}
		if err := healthUnitActive(o.Unit); err != nil {
			return failHealth(err)
		}
		if !o.SkipVersionProbe {
			if err := probeBinaryVersion(dest, job.TargetVersion); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	_ = os.Remove(prevPath)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, prevPath); err != nil {
			// Cross-device fallback
			if err2 := copyFile(dest, prevPath); err2 != nil {
				return fmt.Errorf("preserve previous binary: %w", err)
			}
			_ = os.Remove(dest)
		}
	}
	if err := os.Rename(newPath, dest); err != nil {
		if err2 := copyFile(newPath, dest); err2 != nil {
			// try restore prev
			_ = restorePrevBinary(dest, prevPath)
			return fmt.Errorf("install binary: %w", err)
		}
		_ = os.Remove(newPath)
	}
	_ = os.Chmod(dest, 0o755)
	return nil
}

func restorePrevBinary(dest, prevPath string) error {
	if _, err := os.Stat(prevPath); err != nil {
		return err
	}
	_ = os.Remove(dest)
	if err := os.Rename(prevPath, dest); err != nil {
		return copyFile(prevPath, dest)
	}
	return nil
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o755)
}

func healthUnitActive(unit string) error {
	out, err := exec.Command("systemctl", "is-active", unit).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil || (s != "active" && s != "activating") {
		return fmt.Errorf("health: unit %s is %q", unit, s)
	}
	return nil
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
