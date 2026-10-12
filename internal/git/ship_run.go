package gitstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
)

// ShipOpts controls project ship (PLAN-PROJECT-SHIP).
type ShipOpts struct {
	Project string
	Ref     string
	Tag     string
}

// ShipProject creates a job and runs until done, paused, waiting_external, or failed.
func ShipProject(o ShipOpts) (job *ShipJob, logOut string, err error) {
	p, err := GetProject(o.Project)
	if err != nil {
		return nil, "", err
	}
	ref := o.Ref
	if ref == "" {
		ref = "HEAD"
	}
	tag := strings.TrimSpace(o.Tag)
	if tag != "" && !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	steps := []ShipStep{
		{Name: "test_build", Status: "pending"},
		{Name: "publish", Status: "pending"},
	}
	job, err = CreateShipJob(p.Name, ref, tag, steps)
	if err != nil {
		return nil, "", err
	}
	var logb strings.Builder
	fmt.Fprintf(&logb, "ship job=%s project=%s host=%s ref=%s tag=%s\n", job.ID, p.Name, p.Host, ref, tag)

	if isMacHost(p.Host) {
		if err := MarkShipWaiting(job.ID, "mac"); err != nil {
			return job, logb.String(), err
		}
		fmt.Fprintf(&logb, "waiting_external reason=mac\n  complete: netductor project complete %s --from <dist>\n", job.ID)
		notify.AlertOnce("ship:wait-mac:"+job.ID, fmt.Sprintf("⏸ Ship <code>%s</code> waiting Mac (job <code>%s</code>)", p.Name, job.ID))
		job, _ = GetShipJob(job.ID)
		return job, logb.String(), nil
	}

	return runShipFromCursor(job.ID, &logb)
}

// ResumeShip continues a paused job (does not re-run ok steps).
func ResumeShip(id string) (job *ShipJob, logOut string, err error) {
	j, err := GetShipJob(id)
	if err != nil {
		return nil, "", err
	}
	if j.State != ShipPaused {
		return j, "", fmt.Errorf("job %s state=%s (resume only for paused)", id, j.State)
	}
	var logb strings.Builder
	fmt.Fprintf(&logb, "resume job=%s cursor=%d\n", id, j.Cursor)
	return runShipFromCursor(id, &logb)
}

// CompleteShip attaches external/Mac artifacts and finishes publish.
func CompleteShip(id, fromDir string) (job *ShipJob, logOut string, err error) {
	fromDir = strings.TrimSpace(fromDir)
	if fromDir == "" {
		return nil, "", fmt.Errorf("--from dir required")
	}
	st, err := os.Stat(fromDir)
	if err != nil || !st.IsDir() {
		return nil, "", fmt.Errorf("from dir: %v", err)
	}
	j, err := GetShipJob(id)
	if err != nil {
		return nil, "", err
	}
	if j.State != ShipWaitingExternal && j.State != ShipPaused {
		return j, "", fmt.Errorf("job %s state=%s (complete expects waiting_external or paused)", id, j.State)
	}
	var logb strings.Builder
	fmt.Fprintf(&logb, "complete job=%s from=%s\n", id, fromDir)

	dstArt := ShipJobArtifactsDir(id)
	_ = os.RemoveAll(dstArt)
	if err := copyDir(fromDir, dstArt); err != nil {
		return j, logb.String(), fmt.Errorf("copy artifacts: %w", err)
	}
	fmt.Fprintf(&logb, "artifacts → %s\n", dstArt)

	// mark test_build ok if pending
	j, _ = GetShipJob(id)
	for i := range j.Steps {
		if j.Steps[i].Name == "test_build" && j.Steps[i].Status != "ok" {
			j.Steps[i].Status = "ok"
			j.Steps[i].Outputs = dstArt
		}
	}
	j.Cursor = indexOfStep(j, "publish")
	if j.Cursor < 0 {
		j.Cursor = len(j.Steps) - 1
	}
	_ = SaveShipJob(j)

	return runShipFromCursor(id, &logb)
}

func indexOfStep(j *ShipJob, name string) int {
	for i, s := range j.Steps {
		if s.Name == name {
			return i
		}
	}
	return -1
}

func runShipFromCursor(id string, logb *strings.Builder) (*ShipJob, string, error) {
	j, err := GetShipJob(id)
	if err != nil {
		return nil, logb.String(), err
	}

	if err := TryAcquireVpsSlot(id, ShipRunning); err != nil {
		j.Error = err.Error()
		_ = SaveShipJob(j)
		return j, logb.String(), err
	}

	for {
		j, err = GetShipJob(id)
		if err != nil {
			return nil, logb.String(), err
		}
		if j.Cursor >= len(j.Steps) {
			j.State = ShipDone
			j.VpsSlot = false
			_ = SaveShipJob(j)
			return j, logb.String(), nil
		}
		step := &j.Steps[j.Cursor]
		if step.Status == "ok" || step.Status == "skipped" {
			j.Cursor++
			_ = SaveShipJob(j)
			continue
		}

		step.Status = "running"
		_ = SaveShipJob(j)
		fmt.Fprintf(logb, "==> step %s\n", step.Name)

		var stepErr error
		switch step.Name {
		case "test_build":
			stepErr = runShipTestBuild(j, logb)
		case "publish":
			stepErr = runShipPublish(j, logb)
		default:
			stepErr = fmt.Errorf("unknown step %s", step.Name)
		}

		j, _ = GetShipJob(id)
		if stepErr != nil {
			j.Steps[j.Cursor].Status = "failed"
			j.State = ShipFailed
			j.VpsSlot = false
			j.Error = stepErr.Error()
			j.LogTail = trimTail(logb.String(), 2000)
			_ = SaveShipJob(j)
			notify.AlertOnce("ship:fail:"+id, fmt.Sprintf("🔴 Ship job <code>%s</code> failed at %s", j.Project, step.Name))
			return j, logb.String(), stepErr
		}
		j.Steps[j.Cursor].Status = "ok"
		j.Cursor++
		_ = SaveShipJob(j)

		// pause after successful step if requested
		j, _ = GetShipJob(id)
		if j.PauseRequested {
			j.PauseRequested = false
			j.State = ShipPaused
			j.Reason = "user"
			j.VpsSlot = false
			j.LogTail = trimTail(logb.String(), 2000)
			_ = SaveShipJob(j)
			fmt.Fprintf(logb, "paused after step (cursor=%d)\n", j.Cursor)
			notify.AlertOnce("ship:paused:"+id, fmt.Sprintf("⏸ Ship <code>%s</code> paused (job <code>%s</code>)", j.Project, id))
			return j, logb.String(), nil
		}
	}
}

func runShipTestBuild(j *ShipJob, logb *strings.Builder) error {
	os.Setenv("NETDUCTOR_GIT_BUILD_DIRECT", "1")
	defer os.Unsetenv("NETDUCTOR_GIT_BUILD_DIRECT")
	out, err := runBuildProjectNow(j.Project, j.Ref)
	logb.WriteString(out)
	return err
}

func runShipPublish(j *ShipJob, logb *strings.Builder) error {
	pubName := strings.TrimSpace(j.Tag)
	if pubName == "" {
		pubName = j.ID
	}
	dst := filepath.Join(ArtifactDir(), sanitize(j.Project), sanitize(pubName))
	_ = os.MkdirAll(dst, 0o755)
	logPath := filepath.Join(dst, "ship.log")
	_ = os.WriteFile(logPath, []byte(logb.String()), 0o600)

	srcArt := ShipJobArtifactsDir(j.ID)
	if ents, e := os.ReadDir(srcArt); e == nil {
		for _, ent := range ents {
			_ = copyFileOrDir(filepath.Join(srcArt, ent.Name()), filepath.Join(dst, ent.Name()))
		}
	}
	fmt.Fprintf(logb, "publish → %s\n", dst)

	j2, err := GetShipJob(j.ID)
	if err != nil {
		return err
	}
	j2.PublishTarget = dst
	j2.State = ShipDone
	j2.VpsSlot = false
	if j2.Tag == "" {
		j2.Tag = pubName
	}
	j2.LogTail = trimTail(logb.String(), 2000)
	for i := range j2.Steps {
		if j2.Steps[i].Name == "publish" {
			j2.Steps[i].Status = "ok"
			j2.Steps[i].Outputs = dst
		}
	}
	_ = SaveShipJob(j2)
	notify.AlertOnce("ship:ok:"+j.ID, fmt.Sprintf("✅ Ship <code>%s</code> → <code>%s</code>", j.Project, dst))
	_ = PruneArtifacts(20)
	return nil
}

func trimTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func copyFileOrDir(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return copyDir(src, dst)
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(dst), 0o755)
	return os.WriteFile(dst, b, 0o644)
}

func copyDir(src, dst string) error {
	_ = os.MkdirAll(dst, 0o755)
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else if err := copyFileOrDir(s, d); err != nil {
			return err
		}
	}
	return nil
}
