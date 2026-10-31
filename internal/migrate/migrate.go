// Package migrate runs startup migrations before control-plane Ready (API serve).
package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

// Status of a recorded migration.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Migration is one idempotent upgrade step.
type Migration interface {
	ID() string
	Description() string
	// Required reports whether work is still needed.
	Required(ctx context.Context) (bool, error)
	Apply(ctx context.Context) error
	Verify(ctx context.Context) error
}

type record struct {
	ID          string    `json:"id"`
	Status      Status    `json:"status"`
	Description string    `json:"description,omitempty"`
	AppVersion  string    `json:"app_version,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type journalFile struct {
	Schema     int               `json:"schema"`
	UpdatedAt  time.Time         `json:"updated_at"`
	AppVersion string            `json:"app_version"`
	Records    map[string]record `json:"records"`
}

func journalPath() string {
	return filepath.Join(paths.StateDir(), "migrate", "journal.json")
}

func lockPath() string {
	return filepath.Join(paths.StateDir(), "migrate", "migrate.lock")
}

func durableWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".nd-migrate-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmp := f.Name()
	cleanup := func() { _ = os.Remove(tmp) }
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("chmod temporary file: %w", err)
	}
	n, err := f.Write(data)
	if err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if n != len(data) {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("short write: %d of %d bytes", n, len(data))
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temporary file: %w", err)
	}
	df, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open parent directory: %w", err)
	}
	if err := df.Sync(); err != nil {
		_ = df.Close()
		return fmt.Errorf("sync parent directory: %w", err)
	}
	if err := df.Close(); err != nil {
		return fmt.Errorf("close parent directory: %w", err)
	}
	return nil
}

func loadJournal() (journalFile, error) {
	j := journalFile{Schema: 1, Records: map[string]record{}}
	b, err := os.ReadFile(journalPath())
	if err != nil {
		if os.IsNotExist(err) {
			return j, nil
		}
		return j, err
	}
	if err := json.Unmarshal(b, &j); err != nil {
		return j, fmt.Errorf("migrate journal corrupt: %w", err)
	}
	if j.Schema != 1 {
		return j, fmt.Errorf("migrate journal unsupported schema %d", j.Schema)
	}
	if j.Records == nil {
		return j, fmt.Errorf("migrate journal corrupt: records is null")
	}
	for key, rec := range j.Records {
		if key == "" || rec.ID == "" || key != rec.ID {
			return j, fmt.Errorf("migrate journal corrupt: record key/id mismatch for %q", key)
		}
		switch rec.Status {
		case StatusPending, StatusRunning, StatusCompleted, StatusFailed:
		default:
			return j, fmt.Errorf("migrate journal corrupt: invalid status %q for %s", rec.Status, key)
		}
	}
	return j, nil
}

func durableWriteJournal(j journalFile) error {
	j.Schema = 1
	j.UpdatedAt = time.Now().UTC()
	j.AppVersion = ndver.Release
	dir := filepath.Dir(journalPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	tmp := journalPath() + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("migrate journal open tmp: %w", err)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("migrate journal chmod tmp: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("migrate journal write tmp: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("migrate journal sync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("migrate journal close tmp: %w", err)
	}
	if err := os.Rename(tmp, journalPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	df, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("migrate journal open dir: %w", err)
	}
	if err := df.Sync(); err != nil {
		_ = df.Close()
		return fmt.Errorf("migrate journal sync dir: %w", err)
	}
	if err := df.Close(); err != nil {
		return fmt.Errorf("migrate journal close dir: %w", err)
	}
	return nil
}

// withJournalLock holds the exclusive migrate flock for the duration of fn.
func withJournalLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(lockPath()), 0o700); err != nil {
		return err
	}
	lf, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("migrate lock: %w", err)
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)
	return fn()
}

// Result summarizes one RunAll invocation.
type Result struct {
	Ran       []string `json:"ran"`
	Skipped   []string `json:"skipped"`
	Failed    string   `json:"failed,omitempty"`
	JournalOK bool     `json:"journal_ok"`
}

// DefaultRegistry is the built-in ordered list of migrations.
var DefaultRegistry []Migration

func init() {
	DefaultRegistry = []Migration{
		&ensureStateDirsMigration{},
		&repairStateOnUpgradeMigration{},
		&ensureAptGuardPolicyMigration{},
		&ensureRoleFileMigration{},
		&ensureVPNDirsMigration{},
		&ensureBaselineDirsMigration{},
		&ensureInstalledVersionMigration{},
	}
}

// RunAll acquires a cross-process lock and applies pending migrations in order.
// On failure it returns an error and does not mark Ready — caller should exit.
// After a successful registry run, PostCheck validates minimal host state.
func RunAll(ctx context.Context) (Result, error) {
	res, err := RunRegistry(ctx, DefaultRegistry)
	if err != nil {
		return res, err
	}
	if err := PostCheck(); err != nil {
		return res, err
	}
	return res, nil
}

// RunRegistry runs the given ordered migrations.
func RunRegistry(ctx context.Context, list []Migration) (Result, error) {
	var res Result
	if err := os.MkdirAll(filepath.Dir(lockPath()), 0o700); err != nil {
		return res, err
	}
	lf, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return res, err
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return res, fmt.Errorf("migrate lock: %w", err)
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	j, err := loadJournal()
	if err != nil {
		return res, err
	}
	res.JournalOK = true

	for _, m := range list {
		id := m.ID()
		rec := j.Records[id]
		if rec.Status == StatusCompleted {
			// Still re-check Required for safety after code changes
			need, err := m.Required(ctx)
			if err != nil {
				res.Failed = id
				return res, fmt.Errorf("migrate %s required: %w", id, err)
			}
			if !need {
				res.Skipped = append(res.Skipped, id)
				continue
			}
			// Required again after completed — re-run
		}
		need, err := m.Required(ctx)
		if err != nil {
			res.Failed = id
			return res, fmt.Errorf("migrate %s required: %w", id, err)
		}
		if !need {
			// Record completed if never recorded
			if rec.Status != StatusCompleted {
				now := time.Now().UTC()
				j.Records[id] = record{
					ID: id, Status: StatusCompleted, Description: m.Description(),
					AppVersion: ndver.Release, FinishedAt: now,
				}
				if err := durableWriteJournal(j); err != nil {
					return res, err
				}
			}
			res.Skipped = append(res.Skipped, id)
			continue
		}
		now := time.Now().UTC()
		j.Records[id] = record{
			ID: id, Status: StatusRunning, Description: m.Description(),
			AppVersion: ndver.Release, StartedAt: now,
		}
		if err := durableWriteJournal(j); err != nil {
			return res, fmt.Errorf("migrate journal running: %w", err)
		}
		if err := m.Apply(ctx); err != nil {
			j.Records[id] = record{
				ID: id, Status: StatusFailed, Description: m.Description(),
				AppVersion: ndver.Release, StartedAt: now, FinishedAt: time.Now().UTC(),
				Error: err.Error(),
			}
			_ = durableWriteJournal(j)
			res.Failed = id
			return res, fmt.Errorf("migrate %s apply: %w", id, err)
		}
		if err := m.Verify(ctx); err != nil {
			j.Records[id] = record{
				ID: id, Status: StatusFailed, Description: m.Description(),
				AppVersion: ndver.Release, StartedAt: now, FinishedAt: time.Now().UTC(),
				Error: "verify: " + err.Error(),
			}
			_ = durableWriteJournal(j)
			res.Failed = id
			return res, fmt.Errorf("migrate %s verify: %w", id, err)
		}
		j.Records[id] = record{
			ID: id, Status: StatusCompleted, Description: m.Description(),
			AppVersion: ndver.Release, StartedAt: now, FinishedAt: time.Now().UTC(),
		}
		if err := durableWriteJournal(j); err != nil {
			return res, fmt.Errorf("migrate journal complete: %w", err)
		}
		res.Ran = append(res.Ran, id)
	}
	return res, nil
}

// JournalRecord is a public view of one migration record.
type JournalRecord struct {
	Status     Status `json:"status"`
	AppVersion string `json:"app_version"`
	Error      string `json:"error,omitempty"`
}

// LoadJournalPublic returns id → status for CLI/UI.
func LoadJournalPublic() (map[string]JournalRecord, error) {
	j, err := loadJournal()
	if err != nil {
		return nil, err
	}
	out := map[string]JournalRecord{}
	for id, r := range j.Records {
		out[id] = JournalRecord{Status: r.Status, AppVersion: r.AppVersion, Error: r.Error}
	}
	return out, nil
}

// ClearFailedRecords resets failed entries so the next RunAll can retry Required work.
// Holds the same exclusive lock as RunRegistry to avoid racing the Migration Runner.
func ClearFailedRecords() error {
	return withJournalLock(func() error {
		j, err := loadJournal()
		if err != nil {
			return err
		}
		changed := false
		for id, rec := range j.Records {
			if rec.Status == StatusFailed || rec.Status == StatusRunning {
				delete(j.Records, id)
				changed = true
			}
		}
		if !changed {
			return nil
		}
		return durableWriteJournal(j)
	})
}
