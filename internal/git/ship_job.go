package gitstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Ship job states (PLAN-PROJECT-SHIP).
const (
	ShipQueued          = "queued"
	ShipRunning         = "running"
	ShipPaused          = "paused"
	ShipWaitingExternal = "waiting_external"
	ShipPublishing      = "publishing"
	ShipDone            = "done"
	ShipFailed          = "failed"
	ShipCancelled       = "cancelled"
)

// ShipStep is one pipeline step with status and optional log/output paths.
type ShipStep struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // pending|running|ok|failed|skipped
	Log         string `json:"log,omitempty"`
	Outputs     string `json:"outputs,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	DurationSec int    `json:"duration_sec,omitempty"`
}

// ShipJob is a durable snapshot of a project ship (minimal disk while paused/waiting).
type ShipJob struct {
	ID             string     `json:"id"`
	Project        string     `json:"project"`
	Ref            string     `json:"ref"`
	Tag            string     `json:"tag,omitempty"`
	State          string     `json:"state"`
	Reason         string     `json:"reason,omitempty"` // mac|user|approval|…
	VpsSlot        bool       `json:"vps_slot"`
	Cursor         int        `json:"cursor"`
	Steps          []ShipStep `json:"steps"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	WaitSince      string     `json:"wait_since,omitempty"`
	Error          string     `json:"error,omitempty"`
	LogTail        string     `json:"log_tail,omitempty"`
	PauseRequested bool       `json:"pause_requested,omitempty"`
	CancelRequested bool       `json:"cancel_requested,omitempty"`
	PublishTarget  string     `json:"publish_target,omitempty"` // artifacts path when done
}

var shipMu sync.Mutex

// ShipJobsDir returns /var/lib/netductor/ship-jobs
func ShipJobsDir() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_SHIP_JOBS")); v != "" {
		return v
	}
	return filepath.Join(paths.StateDir(), "ship-jobs")
}

// ShipJobArtifactsDir is per-job output cache (dist etc.).
func ShipJobArtifactsDir(id string) string {
	return filepath.Join(ShipJobsDir(), sanitize(id), "artifacts")
}

func shipJobPath(id string) string {
	return filepath.Join(ShipJobsDir(), sanitize(id)+".json")
}

// ListShipJobs returns all jobs, newest UpdatedAt first.
func ListShipJobs() ([]ShipJob, error) {
	shipMu.Lock()
	defer shipMu.Unlock()
	return listShipJobsUnlocked()
}

func listShipJobsUnlocked() ([]ShipJob, error) {
	dir := ShipJobsDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ShipJob
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var j ShipJob
		if json.Unmarshal(b, &j) != nil || j.ID == "" {
			continue
		}
		out = append(out, j)
	}
	sort.Slice(out, func(i, k int) bool {
		return out[i].UpdatedAt > out[k].UpdatedAt
	})
	return out, nil
}

// GetShipJob loads one job by id.
func GetShipJob(id string) (*ShipJob, error) {
	shipMu.Lock()
	defer shipMu.Unlock()
	return getShipJobUnlocked(id)
}

func getShipJobUnlocked(id string) (*ShipJob, error) {
	id = sanitize(id)
	b, err := os.ReadFile(shipJobPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("ship job not found: %s", id)
		}
		return nil, err
	}
	var j ShipJob
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	return &j, nil
}

// SaveShipJob writes job JSON (creates dir).
func SaveShipJob(j *ShipJob) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	return saveShipJobUnlocked(j)
}

func saveShipJobUnlocked(j *ShipJob) error {
	if j == nil || j.ID == "" {
		return fmt.Errorf("ship job id required")
	}
	j.ID = sanitize(j.ID)
	j.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = os.MkdirAll(ShipJobsDir(), 0o700)
	raw, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(shipJobPath(j.ID), append(raw, '\n'), 0o600)
}

// RemoveShipJob deletes job JSON and per-job artifacts dir.
func RemoveShipJob(id string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	id = sanitize(id)
	j, err := getShipJobUnlocked(id)
	if err != nil {
		return err
	}
	if j.VpsSlot && (j.State == ShipRunning || j.State == ShipPublishing) {
		return fmt.Errorf("cannot remove job %s while holding vps_slot (state=%s); cancel first", id, j.State)
	}
	_ = os.Remove(shipJobPath(id))
	_ = os.RemoveAll(filepath.Join(ShipJobsDir(), id))
	return nil
}

// VpsSlotTaken reports whether any job currently holds the VPS compute slot.
func VpsSlotTaken() bool {
	shipMu.Lock()
	defer shipMu.Unlock()
	list, _ := listShipJobsUnlocked()
	for _, j := range list {
		if j.VpsSlot && (j.State == ShipRunning || j.State == ShipPublishing) {
			return true
		}
	}
	return false
}

// CountShipJobsByState returns counts for doctor/UI.
func CountShipJobsByState() map[string]int {
	shipMu.Lock()
	defer shipMu.Unlock()
	list, _ := listShipJobsUnlocked()
	m := map[string]int{}
	for _, j := range list {
		m[j.State]++
	}
	return m
}

// NewShipJobID builds id project-unix.
func NewShipJobID(project string) string {
	project = sanitize(project)
	return fmt.Sprintf("%s-%d", project, time.Now().UTC().Unix())
}

// CreateShipJob initializes a queued job with default steps (test, build, publish).
func CreateShipJob(project, ref, tag string, steps []ShipStep) (*ShipJob, error) {
	project = sanitize(project)
	if project == "" {
		return nil, fmt.Errorf("project required")
	}
	if ref == "" {
		ref = "HEAD"
	}
	if len(steps) == 0 {
		steps = []ShipStep{
			{Name: "test", Status: "pending"},
			{Name: "build", Status: "pending"},
			{Name: "publish", Status: "pending"},
		}
	}
	j := &ShipJob{
		ID:        NewShipJobID(project),
		Project:   project,
		Ref:       ref,
		Tag:       tag,
		State:     ShipQueued,
		VpsSlot:   false,
		Cursor:    0,
		Steps:     steps,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := SaveShipJob(j); err != nil {
		return nil, err
	}
	return j, nil
}

// MarkShipWaiting moves job to waiting_external and clears vps_slot.
func MarkShipWaiting(id, reason string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	j, err := getShipJobUnlocked(id)
	if err != nil {
		return err
	}
	j.State = ShipWaitingExternal
	j.Reason = reason
	j.VpsSlot = false
	j.WaitSince = time.Now().UTC().Format(time.RFC3339)
	return saveShipJobUnlocked(j)
}

// TryAcquireVpsSlot sets running/publishing + vps_slot if no other holder.
func TryAcquireVpsSlot(id, state string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	if state != ShipRunning && state != ShipPublishing {
		return fmt.Errorf("acquire state must be running or publishing")
	}
	list, err := listShipJobsUnlocked()
	if err != nil {
		return err
	}
	for _, o := range list {
		if o.ID != sanitize(id) && o.VpsSlot && (o.State == ShipRunning || o.State == ShipPublishing) {
			return fmt.Errorf("vps slot held by job %s (%s)", o.ID, o.State)
		}
	}
	j, err := getShipJobUnlocked(id)
	if err != nil {
		return err
	}
	j.State = state
	j.VpsSlot = true
	j.WaitSince = ""
	return saveShipJobUnlocked(j)
}

// ReleaseVpsSlot clears vps_slot without changing logical wait state caller sets after.
func ReleaseVpsSlot(id string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	j, err := getShipJobUnlocked(id)
	if err != nil {
		return err
	}
	j.VpsSlot = false
	return saveShipJobUnlocked(j)
}

// FormatShipJobsText is human listing for CLI.
func FormatShipJobsText(jobs []ShipJob) string {
	if len(jobs) == 0 {
		return "(no ship jobs)\n"
	}
	var b strings.Builder
	b.WriteString("id\tproject\tstate\tslot\ttag\tref\tupdated\n")
	for _, j := range jobs {
		slot := "0"
		if j.VpsSlot {
			slot = "1"
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			j.ID, j.Project, j.State, slot, j.Tag, j.Ref, j.UpdatedAt)
	}
	return b.String()
}

// RequestShipPause sets pause_requested; applied after the current step finishes.
func RequestShipPause(id string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	j, err := getShipJobUnlocked(id)
	if err != nil {
		return err
	}
	if j.State == ShipDone || j.State == ShipFailed {
		return fmt.Errorf("job %s is %s", id, j.State)
	}
	j.PauseRequested = true
	return saveShipJobUnlocked(j)
}

// RequestShipCancel marks job cancelled. For running builds, kills CI container (grace via docker).
func RequestShipCancel(id string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	j, err := getShipJobUnlocked(id)
	if err != nil {
		return err
	}
	if j.State == ShipDone || j.State == ShipFailed || j.State == ShipCancelled {
		return fmt.Errorf("job %s is %s", id, j.State)
	}
	j.CancelRequested = true
	j.State = ShipCancelled
	j.VpsSlot = false
	j.Error = "cancelled"
	j.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveShipJobUnlocked(j); err != nil {
		return err
	}
	// Best-effort stop of isolated CI container for this ship job.
	killCIContainer("ship-" + sanitize(id))
	return nil
}

// shouldPauseAfterStep returns true if pause was requested (clears flag when applying).


// ShipJobLogPath is the full log file for a job (appended during run).
func ShipJobLogPath(id string) string {
	return filepath.Join(ShipJobsDir(), sanitize(id), "ship.log")
}

// AppendShipLog writes to job log file.
func AppendShipLog(id, chunk string) {
	if chunk == "" {
		return
	}
	p := ShipJobLogPath(id)
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(chunk)
	if !strings.HasSuffix(chunk, "\n") {
		_, _ = f.WriteString("\n")
	}
}

// ReadShipLog returns full or tailed log.
func ReadShipLog(id string, maxBytes int) (string, error) {
	b, err := os.ReadFile(ShipJobLogPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			j, e2 := GetShipJob(id)
			if e2 == nil && j.LogTail != "" {
				return j.LogTail, nil
			}
			return "", err
		}
		return "", err
	}
	if maxBytes > 0 && len(b) > maxBytes {
		return string(b[len(b)-maxBytes:]), nil
	}
	return string(b), nil
}
