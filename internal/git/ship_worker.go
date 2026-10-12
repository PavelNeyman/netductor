package gitstore

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
)

var (
	shipWorkerOnce sync.Once
)

// StartShipWorker processes queued ship jobs one at a time (respects vps_slot).
func StartShipWorker() {
	if os.Getenv("NETDUCTOR_DISABLE_SHIP_WORKER") == "1" {
		return
	}
	shipWorkerOnce.Do(func() {
		go shipWorkerLoop()
	})
}

func shipWorkerLoop() {
	for {
		job, ok := claimNextShipJob()
		if !ok {
			time.Sleep(2 * time.Second)
			continue
		}
		var logb strings.Builder
		fmt.Fprintf(&logb, "worker ship job=%s project=%s\n", job.ID, job.Project)
		_, _, err := runShipFromCursor(job.ID, &logb)
		if err != nil {
			notify.AlertOnce("ship:worker-fail:"+job.ID, fmt.Sprintf("🔴 Ship worker <code>%s</code>: %v", job.Project, err))
		}
	}
}

func claimNextShipJob() (ShipJob, bool) {
	shipMu.Lock()
	defer shipMu.Unlock()
	// do not claim if slot held
	list, _ := listShipJobsUnlocked()
	for _, j := range list {
		if j.VpsSlot && (j.State == ShipRunning || j.State == ShipPublishing) {
			return ShipJob{}, false
		}
	}
	for i := range list {
		if list[i].State == ShipQueued {
			list[i].State = ShipRunning
			list[i].VpsSlot = true
			list[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			_ = saveShipJobUnlocked(&list[i])
			return list[i], true
		}
	}
	return ShipJob{}, false
}

// EnqueueShipJob creates queued job for worker (VPS). Mac → waiting_external immediately.
func EnqueueShipJob(o ShipOpts) (*ShipJob, error) {
	p, err := GetProject(o.Project)
	if err != nil {
		return nil, err
	}
	ref := o.Ref
	if ref == "" {
		ref = "HEAD"
	}
	tag := strings.TrimSpace(o.Tag)
	if tag == "" {
		tag = NextProjectTag(p.Name)
	} else if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	steps := []ShipStep{
		{Name: "test_build", Status: "pending"},
		{Name: "publish", Status: "pending"},
	}
	job, err := CreateShipJob(p.Name, ref, tag, steps)
	if err != nil {
		return nil, err
	}
	if isMacHost(p.Host) {
		_ = MarkShipWaiting(job.ID, "mac")
		notify.AlertOnce("ship:wait-mac:"+job.ID, fmt.Sprintf("⏸ Ship <code>%s</code> waiting Mac (job <code>%s</code>)", p.Name, job.ID))
		return GetShipJob(job.ID)
	}
	StartShipWorker()
	return job, nil
}

// ShipAllVPS enqueues ship for each non-mac project.
func ShipAllVPS() (queued, skipped []string, err error) {
	list, err := LoadProjects()
	if err != nil {
		return nil, nil, err
	}
	for _, p := range list {
		if isMacHost(p.Host) {
			skipped = append(skipped, p.Name+":mac")
			continue
		}
		job, e := EnqueueShipJob(ShipOpts{Project: p.Name})
		if e != nil {
			skipped = append(skipped, p.Name+":"+e.Error())
			continue
		}
		queued = append(queued, job.ID)
	}
	return queued, skipped, nil
}
