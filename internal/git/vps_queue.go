package gitstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// VpsBuildJob is a sequential CI job on primary (one worker, no parallel builds).
type VpsBuildJob struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Ref       string `json:"ref"`
	Kind      string `json:"kind"`   // build|sync
	Status    string `json:"status"` // pending|running|done|failed|cancelled
	CreatedAt string `json:"created_at"`
	StartedAt string `json:"started_at,omitempty"`
	DoneAt    string `json:"done_at,omitempty"`
	Error     string `json:"error,omitempty"`
	LogTail   string `json:"log_tail,omitempty"`
}

var (
	vpsMu       sync.Mutex
	vpsWorkerOn sync.Once
)

func vpsQueuePath() string {
	return filepath.Join(paths.StateDir(), "git-vps-queue.json")
}

func loadVpsQueue() []VpsBuildJob {
	b, err := os.ReadFile(vpsQueuePath())
	if err != nil {
		return nil
	}
	var list []VpsBuildJob
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return list
}

func saveVpsQueue(list []VpsBuildJob) error {
	_ = os.MkdirAll(filepath.Dir(vpsQueuePath()), 0o700)
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vpsQueuePath(), append(raw, '\n'), 0o600)
}

// StartVpsBuildWorker runs a single background worker (safe to call many times).
func StartVpsBuildWorker() {
	vpsWorkerOn.Do(func() {
		go vpsBuildWorkerLoop()
	})
}

func vpsBuildWorkerLoop() {
	for {
		job, ok := claimNextVpsJob()
		if !ok {
			time.Sleep(2 * time.Second)
			continue
		}
		runClaimedVpsJob(job)
	}
}

func claimNextVpsJob() (VpsBuildJob, bool) {
	vpsMu.Lock()
	defer vpsMu.Unlock()
	list := loadVpsQueue()
	for i := range list {
		if list[i].Status == "pending" {
			list[i].Status = "running"
			list[i].StartedAt = time.Now().UTC().Format(time.RFC3339)
			_ = saveVpsQueue(list)
			return list[i], true
		}
	}
	return VpsBuildJob{}, false
}

func runClaimedVpsJob(job VpsBuildJob) {
	var out string
	var err error
	switch job.Kind {
	case "sync":
		out, err = SyncProject(job.Project)
	default:
		out, err = runBuildProjectNow(job.Project, job.Ref)
	}
	tail := out
	if len(tail) > 4000 {
		tail = tail[len(tail)-4000:]
	}
	vpsMu.Lock()
	defer vpsMu.Unlock()
	list := loadVpsQueue()
	for i := range list {
		if list[i].ID != job.ID {
			continue
		}
		list[i].DoneAt = time.Now().UTC().Format(time.RFC3339)
		list[i].LogTail = tail
		if err != nil {
			list[i].Status = "failed"
			list[i].Error = err.Error()
			notify.AlertOnce("git:vps-build-fail:"+job.Project, fmt.Sprintf("🔴 VPS build <code>%s</code> failed: %s", job.Project, err.Error()))
		} else {
			list[i].Status = "done"
			list[i].Error = ""
			notify.AlertOnce("git:vps-build-ok:"+job.Project, fmt.Sprintf("✅ VPS build <code>%s</code> ok", job.Project))
		}
		break
	}
	// keep last 80
	if len(list) > 80 {
		list = list[len(list)-80:]
	}
	_ = saveVpsQueue(list)
}

// EnqueueVpsJob adds a build/sync job. Dedupes identical pending project+kind+ref.
func EnqueueVpsJob(project, ref, kind string) (VpsBuildJob, error) {
	StartVpsBuildWorker()
	vpsMu.Lock()
	defer vpsMu.Unlock()
	project = sanitize(project)
	if project == "" {
		return VpsBuildJob{}, fmt.Errorf("empty project")
	}
	if ref == "" {
		ref = "HEAD"
	}
	if kind == "" {
		kind = "build"
	}
	list := loadVpsQueue()
	for _, j := range list {
		if j.Status == "pending" && j.Project == project && j.Kind == kind && j.Ref == ref {
			return j, nil
		}
		if j.Status == "running" && j.Project == project && j.Kind == kind {
			return j, nil
		}
	}
	id := fmt.Sprintf("%s-%s-%d", kind, project, time.Now().UTC().UnixNano())
	job := VpsBuildJob{
		ID:        id,
		Project:   project,
		Ref:       ref,
		Kind:      kind,
		Status:    "pending",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	list = append(list, job)
	if err := saveVpsQueue(list); err != nil {
		return job, err
	}
	return job, nil
}

// ListVpsQueue returns jobs; pendingOnly filters to pending|running.
func ListVpsQueue(pendingOnly bool) []VpsBuildJob {
	vpsMu.Lock()
	defer vpsMu.Unlock()
	list := loadVpsQueue()
	if !pendingOnly {
		return list
	}
	var out []VpsBuildJob
	for _, j := range list {
		if j.Status == "pending" || j.Status == "running" {
			out = append(out, j)
		}
	}
	return out
}

// WaitVpsJob blocks until job leaves pending|running or timeout.
func WaitVpsJob(id string, timeout time.Duration) (VpsBuildJob, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		vpsMu.Lock()
		list := loadVpsQueue()
		vpsMu.Unlock()
		for _, j := range list {
			if j.ID != id {
				continue
			}
			if j.Status == "done" || j.Status == "failed" || j.Status == "cancelled" {
				if j.Status == "failed" {
					return j, fmt.Errorf("%s", j.Error)
				}
				return j, nil
			}
			break
		}
		time.Sleep(time.Second)
	}
	return VpsBuildJob{}, fmt.Errorf("timeout waiting for job %s", id)
}

// CancelVpsJob marks a pending job cancelled.
func CancelVpsJob(id string) error {
	vpsMu.Lock()
	defer vpsMu.Unlock()
	list := loadVpsQueue()
	for i := range list {
		if list[i].ID == id && list[i].Status == "pending" {
			list[i].Status = "cancelled"
			list[i].DoneAt = time.Now().UTC().Format(time.RFC3339)
			return saveVpsQueue(list)
		}
	}
	return fmt.Errorf("vps queue job not found or not pending: %s", id)
}

// EnqueueBuildAll queues build for each project (vpsOnly skips mac).
func EnqueueBuildAll(vpsOnly bool) (jobs []VpsBuildJob, skipped []string) {
	list, err := LoadProjects()
	if err != nil {
		return nil, []string{err.Error()}
	}
	for _, p := range list {
		host := strings.TrimSpace(p.Host)
		if host == "" {
			host = "vps"
		}
		if vpsOnly && strings.EqualFold(host, "mac") {
			skipped = append(skipped, p.Name)
			continue
		}
		if strings.EqualFold(host, "mac") {
			_, _ = EnqueueMacBuild(p.Name, "HEAD")
			skipped = append(skipped, p.Name+"→mac-queue")
			continue
		}
		j, err := EnqueueVpsJob(p.Name, "HEAD", "build")
		if err != nil {
			skipped = append(skipped, p.Name+":"+err.Error())
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, skipped
}

// EnqueueSyncAll queues sync for every project (still sequential via worker).
func EnqueueSyncAll() (jobs []VpsBuildJob, errs []string) {
	list, err := LoadProjects()
	if err != nil {
		return nil, []string{err.Error()}
	}
	for _, p := range list {
		j, err := EnqueueVpsJob(p.Name, "HEAD", "sync")
		if err != nil {
			errs = append(errs, p.Name+":"+err.Error())
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, errs
}
