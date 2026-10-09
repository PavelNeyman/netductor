package gitstore

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/gha"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Project is a registered upstream (usually GitHub) mirrored into bare git.
// Build order: host=mac → queue; explicit Workflow → .github/workflows/<Name>.yml → pipeline → detect.
type Project struct {
	Name         string `json:"name"`
	Upstream     string `json:"upstream"` // https://github.com/org/repo.git
	Workflow     string `json:"workflow,omitempty"` // relative path e.g. .github/workflows/ci.yml
	Pipeline     string `json:"pipeline,omitempty"` // shell pipeline name under PipelineDir
	BuildOnFetch bool   `json:"build_on_fetch,omitempty"`
	// Host: "vps" (default, build on primary) or "mac" (op builds, release import only).
	Host         string `json:"host,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

func projectsPath() string {
	if v := os.Getenv("NETDUCTOR_GIT_PROJECTS"); v != "" {
		return v
	}
	return filepath.Join(paths.StateDir(), "git-projects.json")
}

func LoadProjects() ([]Project, error) {
	b, err := os.ReadFile(projectsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []Project
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func saveProjects(list []Project) error {
	_ = os.MkdirAll(filepath.Dir(projectsPath()), 0o755)
	b, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(projectsPath(), append(b, '\n'), 0o600)
}

func GetProject(name string) (*Project, error) {
	name = sanitize(name)
	list, err := LoadProjects()
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Name == name {
			return &list[i], nil
		}
	}
	return nil, fmt.Errorf("project not found: %s", name)
}

// AddProject registers upstream, ensures bare repo + origin, optional initial fetch.
func AddProject(p Project, fetch bool) error {
	p.Name = sanitize(p.Name)
	p.Upstream = strings.TrimSpace(p.Upstream)
	if p.Name == "" || p.Upstream == "" {
		return fmt.Errorf("name and upstream required")
	}
	if !strings.Contains(p.Upstream, "://") && !strings.HasPrefix(p.Upstream, "git@") {
		// allow org/repo → github
		if strings.Count(p.Upstream, "/") == 1 {
			p.Upstream = "https://github.com/" + p.Upstream + ".git"
		} else {
			return fmt.Errorf("upstream must be URL or github org/repo")
		}
	}
	list, err := LoadProjects()
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].Name == p.Name {
			if strings.TrimSpace(p.Workflow) == "" {
				p.Workflow = ".github/workflows/" + p.Name + ".yml"
			}
			if strings.TrimSpace(p.Host) == "" {
				p.Host = list[i].Host
				if p.Host == "" {
					p.Host = "vps"
				}
			}
			list[i] = p
			list[i].CreatedAt = list[i].CreatedAt
			if list[i].CreatedAt == "" {
				list[i].CreatedAt = time.Now().UTC().Format(time.RFC3339)
			}
			if err := saveProjects(list); err != nil {
				return err
			}
			_, err := MirrorEnsure(p.Name, p.Upstream)
			return err
		}
	}
	if strings.TrimSpace(p.Workflow) == "" {
		p.Workflow = ".github/workflows/" + p.Name + ".yml"
	}
	if strings.TrimSpace(p.Host) == "" {
		p.Host = "vps"
	}
	p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	list = append(list, p)
	if err := saveProjects(list); err != nil {
		return err
	}
	if _, err := MirrorEnsure(p.Name, p.Upstream); err != nil {
		return err
	}
	if fetch {
		_, err := MirrorFetch(p.Name)
		return err
	}
	return nil
}

func RemoveProject(name string) error {
	name = sanitize(name)
	list, err := LoadProjects()
	if err != nil {
		return err
	}
	out := list[:0]
	found := false
	for _, p := range list {
		if p.Name == name {
			found = true
			continue
		}
		out = append(out, p)
	}
	if !found {
		return fmt.Errorf("project not found: %s", name)
	}
	return saveProjects(out)
}

// SyncProject fetches upstream into bare mirror.
func SyncProject(name string) (string, error) {
	p, err := GetProject(name)
	if err != nil {
		return "", err
	}
	if _, err := MirrorEnsure(p.Name, p.Upstream); err != nil {
		return "", err
	}
	out, err := MirrorFetch(p.Name)
	if err != nil {
		return out, err
	}
	if p.BuildOnFetch {
		blog, berr := BuildProject(p.Name, "")
		out += "\n" + blog
		if berr != nil {
			return out, berr
		}
	}
	return out, nil
}

// BuildProject checks out ref (default HEAD/main/master) and runs workflow or pipeline.
func BuildProject(name, ref string) (string, error) {
	p, err := GetProject(name)
	if err != nil {
		return "", err
	}
	work := filepath.Join(paths.StateDir(), "ci", "project-"+p.Name+"-"+time.Now().UTC().Format("150405"))
	_ = os.RemoveAll(work)
	defer func() { _ = os.RemoveAll(work) }()

	if ref == "" {
		ref = "HEAD"
		// prefer main/master tag tip
		if tags, _ := ListTags(p.Name); len(tags) > 0 {
			// stay on HEAD of default branch via clone
		}
	}
	if err := checkoutRef(p.Name, ref, work); err != nil {
		return "", err
	}

	var log strings.Builder
	fmt.Fprintf(&log, "project %s ref=%s work=%s host=%s\n", p.Name, ref, work, p.Host)

	// host=mac: never compile on VPS — enqueue + notify operator
	if isMacHost(p.Host) {
		job, err := EnqueueMacBuild(p.Name, ref)
		fmt.Fprintf(&log, "mac-queue id=%s status=%s\ncli: %s\n", job.ID, job.Status, job.CLI)
		if err != nil {
			return log.String(), err
		}
		return log.String(), nil
	}

	// 1) explicit workflow
	if wf := strings.TrimSpace(p.Workflow); wf != "" {
		data, err := os.ReadFile(filepath.Join(work, wf))
		if err != nil {
			return log.String(), fmt.Errorf("workflow %s: %w", wf, err)
		}
		w, err := gha.Parse(data)
		if err != nil {
			return log.String(), err
		}
		out, err := gha.Run(w, work, nil)
		log.WriteString(out)
		if err != nil {
			notify.AlertOnce("git:build-fail:"+p.Name, fmt.Sprintf("🔴 Project build <code>%s</code> workflow failed", p.Name))
			return log.String(), err
		}
		notify.AlertOnce("git:build-ok:"+p.Name, fmt.Sprintf("✅ Project build <code>%s</code> ok (workflow)", p.Name))
		return log.String(), nil
	}

	// 2) exact .github/workflows/<Name>.yml (project name = repo name)
	if path, ferr := gha.ResolveProjectWorkflow(work, p.Name); ferr == nil {
		data, rerr := os.ReadFile(path)
		if rerr == nil {
			if w, perr := gha.Parse(data); perr == nil {
				fmt.Fprintf(&log, "using %s\n", path)
				out, err := gha.Run(w, work, nil)
				log.WriteString(out)
				if err != nil {
					notify.AlertOnce("git:build-fail:"+p.Name, fmt.Sprintf("🔴 Project build <code>%s</code> failed", p.Name))
					return log.String(), err
				}
				notify.AlertOnce("git:build-ok:"+p.Name, fmt.Sprintf("✅ Project build <code>%s</code> ok", p.Name))
				return log.String(), nil
			}
		}
	} else {
		fmt.Fprintf(&log, "workflow resolve: %v\n", ferr)
	}

	// 3) shell pipeline
	if pipe := strings.TrimSpace(p.Pipeline); pipe != "" {
		out, err := RunPipeline(p.Name, pipe)
		log.WriteString(out)
		if err != nil {
			notify.AlertOnce("git:build-fail:"+p.Name, fmt.Sprintf("🔴 Project pipeline <code>%s/%s</code> failed", p.Name, pipe))
			return log.String(), err
		}
		notify.AlertOnce("git:build-ok:"+p.Name, fmt.Sprintf("✅ Project pipeline <code>%s/%s</code> ok", p.Name, pipe))
		return log.String(), nil
	}

	// 4) default: workflow empty path via RunWorkflow (same as git workflow)
	out, err := RunWorkflow(p.Name, "")
	log.WriteString(out)
	if err != nil {
		notify.AlertOnce("git:build-fail:"+p.Name, fmt.Sprintf("🔴 Project build <code>%s</code>: %v", p.Name, err))
		return log.String(), err
	}
	notify.AlertOnce("git:build-ok:"+p.Name, fmt.Sprintf("✅ Project build <code>%s</code> ok", p.Name))
	return log.String(), nil
}

func checkoutRef(name, ref, workDir string) error {
	dir, err := repoDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	if ref == "" {
		ref = "HEAD"
	}
	cmd := exec.Command("git", "--git-dir="+dir, "--work-tree="+workDir, "checkout", "-f", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout %s: %w (%s)", ref, err, strings.TrimSpace(string(out)))
	}
	return nil
}


// ProjectDetail is card payload for UI (thin clients).
type ProjectDetail struct {
	Project
	Tags      []string `json:"tags"`
	Artifacts []string `json:"artifacts"`
	BarePath  string   `json:"bare_path,omitempty"`
	Note      string   `json:"note,omitempty"`
}

func ProjectDetailOf(name string) (*ProjectDetail, error) {
	p, err := GetProject(name)
	if err != nil {
		return nil, err
	}
	d := &ProjectDetail{Project: *p}
	if d.Host == "" {
		d.Host = "vps"
	}
	if dir, err := repoDir(p.Name); err == nil {
		d.BarePath = dir
	}
	d.Tags, _ = ListTags(p.Name)
	if len(d.Tags) > 12 {
		d.Tags = d.Tags[:12]
	}
	d.Artifacts, _ = ListArtifacts(p.Name)
	if len(d.Artifacts) > 20 {
		d.Artifacts = d.Artifacts[len(d.Artifacts)-20:]
	}
	if d.Host == "mac" {
		d.Note = "host=mac: build on operator; import artifacts via release import / scp"
	} else if d.BuildOnFetch {
		d.Note = "build_on_fetch: SyncProject runs Build after fetch (heavy — use sparingly)"
	}
	return d, nil
}
