package gitstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
)

// ShipOpts controls project ship (S1+).
type ShipOpts struct {
	Project string
	Ref     string
	Tag     string // empty → caller may set; optional for artifacts publish name
}

// ShipProject runs full cycle for a registered project (PLAN S1).
// host=mac → job waiting_external (no VPS slot held).
// host=vps → acquire slot, run existing build path, publish logs/outputs under git-artifacts.
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
		job.Steps[0].Status = "pending"
		_ = SaveShipJob(job)
		if err := MarkShipWaiting(job.ID, "mac"); err != nil {
			return job, logb.String(), err
		}
		fmt.Fprintf(&logb, "waiting_external reason=mac — build on operator, then: netductor project complete %s --from <dist>\n", job.ID)
		notify.AlertOnce("ship:wait-mac:"+job.ID, fmt.Sprintf("⏸ Ship <code>%s</code> waiting Mac (job <code>%s</code>)", p.Name, job.ID))
		job, _ = GetShipJob(job.ID)
		return job, logb.String(), nil
	}

	if err := TryAcquireVpsSlot(job.ID, ShipRunning); err != nil {
		job.State = ShipFailed
		job.Error = err.Error()
		_ = SaveShipJob(job)
		return job, logb.String(), err
	}
	job, _ = GetShipJob(job.ID)
	job.Steps[0].Status = "running"
	_ = SaveShipJob(job)

	// Run in-process build (holds ship slot; do not also queue vps worker for same work)
	os.Setenv("NETDUCTOR_GIT_BUILD_DIRECT", "1")
	out, buildErr := runBuildProjectNow(p.Name, ref)
	logb.WriteString(out)
	os.Unsetenv("NETDUCTOR_GIT_BUILD_DIRECT")

	job, _ = GetShipJob(job.ID)
	if buildErr != nil {
		job.Steps[0].Status = "failed"
		job.State = ShipFailed
		job.VpsSlot = false
		job.Error = buildErr.Error()
		job.LogTail = trimTail(logb.String(), 2000)
		_ = SaveShipJob(job)
		notify.AlertOnce("ship:fail:"+job.ID, fmt.Sprintf("🔴 Ship <code>%s</code> failed", p.Name))
		return job, logb.String(), buildErr
	}
	job.Steps[0].Status = "ok"
	job.Cursor = 1
	_ = SaveShipJob(job)

	// publish: store log + any dist under artifacts/<project>/<tag|jobid>/
	pubName := tag
	if pubName == "" {
		pubName = job.ID
	}
	dst := filepath.Join(ArtifactDir(), sanitize(p.Name), sanitize(pubName))
	_ = os.MkdirAll(dst, 0o755)
	logPath := filepath.Join(dst, "ship.log")
	_ = os.WriteFile(logPath, []byte(logb.String()), 0o600)
	// copy job-local artifacts if any
	srcArt := ShipJobArtifactsDir(job.ID)
	if ents, e := os.ReadDir(srcArt); e == nil {
		for _, ent := range ents {
			_ = copyFileOrDir(filepath.Join(srcArt, ent.Name()), filepath.Join(dst, ent.Name()))
		}
	}
	// best-effort: common dist under state ci leftovers is already deleted; workflow may have written to work — gone
	fmt.Fprintf(&logb, "publish → %s\n", dst)
	job.Steps[1].Status = "ok"
	job.Steps[1].Outputs = dst
	job.State = ShipDone
	job.VpsSlot = false
	job.Cursor = 2
	job.LogTail = trimTail(logb.String(), 2000)
	if job.Tag == "" {
		job.Tag = pubName
	}
	_ = SaveShipJob(job)
	notify.AlertOnce("ship:ok:"+job.ID, fmt.Sprintf("✅ Ship <code>%s</code> → <code>%s</code>", p.Name, dst))
	_ = PruneArtifacts(20)
	return job, logb.String(), nil
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
		} else {
			if err := copyFileOrDir(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

