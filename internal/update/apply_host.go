package update

import (
	"fmt"
	"os"
	"os/exec"
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
	// SkipDownload is for tests.
	SkipDownload bool
	// AfterDownload optional hook (tests).
	AfterDownload func() error
}

// DefaultHostRegistry is process-local; CLI uses it to prevent concurrent host applies.
var DefaultHostRegistry = NewJobRegistry()

// ApplyHostComponent runs download → verify (inside DownloadReleaseAsset) → optional restart → health.
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
		// Allow offline tests to inject AfterDownload without network when SkipDownload.
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
	if !o.SkipDownload {
		if err := DownloadNamedAsset(tag, res.Name, o.DestPath); err != nil {
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
	// DownloadNamedAsset already verified SHA when not skip-verify; treat presence of dest as install done.
	if !o.SkipDownload {
		if st, err := os.Stat(o.DestPath); err != nil || st.Size() == 0 {
			j, _ := mark(StateFailed, "install: destination missing or empty")
			return j, fmt.Errorf("update: destination missing or empty")
		}
	}
	if _, err := mark(StateInstalling, ""); err != nil {
		return cur, err
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
			j, _ := mark(StateFailed, err.Error())
			return j, err
		}
	} else {
		// No restart: still require file in place before succeeded.
		if _, err := mark(StateHealthChecking, ""); err != nil {
			return cur, err
		}
	}
	j, err := mark(StateSucceeded, "")
	return j, err
}

func healthUnitActive(unit string) error {
	out, err := exec.Command("systemctl", "is-active", unit).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil || (s != "active" && s != "activating") {
		return fmt.Errorf("health: unit %s is %q", unit, s)
	}
	return nil
}
