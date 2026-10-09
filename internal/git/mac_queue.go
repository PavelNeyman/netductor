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

// MacBuildJob is a pending host=mac build (VPS does not compile).
type MacBuildJob struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Ref       string `json:"ref"`
	Status    string `json:"status"` // pending|done|cancelled
	CreatedAt string `json:"created_at"`
	DoneAt    string `json:"done_at,omitempty"`
	CLI       string `json:"cli"` // ready command for operator Mac
	Note      string `json:"note,omitempty"`
}

var macMu sync.Mutex

func macQueuePath() string {
	return filepath.Join(paths.StateDir(), "git-mac-queue.json")
}

func loadMacQueue() []MacBuildJob {
	b, err := os.ReadFile(macQueuePath())
	if err != nil {
		return nil
	}
	var list []MacBuildJob
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return list
}

func saveMacQueue(list []MacBuildJob) error {
	_ = os.MkdirAll(filepath.Dir(macQueuePath()), 0o700)
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(macQueuePath(), append(raw, '\n'), 0o600)
}

// EnqueueMacBuild records a job and notifies all UI (TG). Returns job id.
func EnqueueMacBuild(project, ref string) (MacBuildJob, error) {
	macMu.Lock()
	defer macMu.Unlock()
	if ref == "" {
		ref = "HEAD"
	}
	project = sanitize(project)
	list := loadMacQueue()
	// dedupe pending same project+ref
	for _, j := range list {
		if j.Status == "pending" && j.Project == project && j.Ref == ref {
			return j, nil
		}
	}
	id := fmt.Sprintf("%s-%d", project, time.Now().UTC().Unix())
	cli := fmt.Sprintf("netductor-op project build %s --ref %s", project, ref)
	job := MacBuildJob{
		ID:        id,
		Project:   project,
		Ref:       ref,
		Status:    "pending",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		CLI:       cli,
		Note:      "host=mac — build on operator Mac, then release import / scp to VPS",
	}
	list = append(list, job)
	// keep last 50
	if len(list) > 50 {
		list = list[len(list)-50:]
	}
	if err := saveMacQueue(list); err != nil {
		return job, err
	}
	msg := fmt.Sprintf(
		"🔨 <b>Build required</b>: <code>%s</code> @ <code>%s</code>\nhost=mac · queue_id=<code>%s</code>\nCLI:\n<pre>%s</pre>",
		project, ref, id, cli,
	)
	notify.AlertOnce("git:mac-build:"+id, msg)
	_ = notify.Telegram(msg) // also try immediate (AlertOnce may batch)
	return job, nil
}

// ListMacQueue returns jobs; pendingOnly filters.
func ListMacQueue(pendingOnly bool) []MacBuildJob {
	macMu.Lock()
	defer macMu.Unlock()
	list := loadMacQueue()
	if !pendingOnly {
		return list
	}
	var out []MacBuildJob
	for _, j := range list {
		if j.Status == "pending" {
			out = append(out, j)
		}
	}
	return out
}

// CompleteMacBuild marks job done (after operator imported artifacts).
func CompleteMacBuild(id string) error {
	macMu.Lock()
	defer macMu.Unlock()
	list := loadMacQueue()
	found := false
	for i := range list {
		if list[i].ID == id || (list[i].Project == id && list[i].Status == "pending") {
			list[i].Status = "done"
			list[i].DoneAt = time.Now().UTC().Format(time.RFC3339)
			found = true
			notify.ClearAlert("git:mac-build:" + list[i].ID)
		}
	}
	if !found {
		return fmt.Errorf("mac queue job not found: %s", id)
	}
	return saveMacQueue(list)
}

// CancelMacBuild marks cancelled.
func CancelMacBuild(id string) error {
	macMu.Lock()
	defer macMu.Unlock()
	list := loadMacQueue()
	found := false
	for i := range list {
		if list[i].ID == id {
			list[i].Status = "cancelled"
			list[i].DoneAt = time.Now().UTC().Format(time.RFC3339)
			found = true
			notify.ClearAlert("git:mac-build:" + list[i].ID)
		}
	}
	if !found {
		return fmt.Errorf("mac queue job not found: %s", id)
	}
	return saveMacQueue(list)
}

func isMacHost(h string) bool {
	return strings.EqualFold(strings.TrimSpace(h), "mac")
}
