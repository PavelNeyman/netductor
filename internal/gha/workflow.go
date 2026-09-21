// Package gha runs a minimal GitHub Actions–compatible subset:
// jobs.*.steps with `run:` (shell), optional job container:.
// Build steps run in containers by default (see internal/ci).
package gha

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ci"
	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name string         `yaml:"name"`
	On   any            `yaml:"on"`
	Jobs map[string]Job `yaml:"jobs"`
}

type Job struct {
	RunsOn    string            `yaml:"runs-on"`
	Container any               `yaml:"container"` // string or {image: ...}
	Steps     []Step            `yaml:"steps"`
	Env       map[string]string `yaml:"env"`
}

type Step struct {
	Name             string            `yaml:"name"`
	Run              string            `yaml:"run"`
	Uses             string            `yaml:"uses"`
	WorkingDirectory string            `yaml:"working-directory"`
	Env              map[string]string `yaml:"env"`
	Shell            string            `yaml:"shell"`
	ID               string            `yaml:"id"`
}

func Parse(data []byte) (*Workflow, error) {
	var w Workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if len(w.Jobs) == 0 {
		return nil, fmt.Errorf("no jobs in workflow")
	}
	return &w, nil
}

func containerImage(c any) string {
	switch v := c.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if img, ok := v["image"].(string); ok {
			return strings.TrimSpace(img)
		}
	case map[any]any:
		if img, ok := v["image"].(string); ok {
			return strings.TrimSpace(img)
		}
	}
	return ""
}

// Run executes jobs in sorted name order; only steps with non-empty run:.
// workDir is the checkout / work tree root. Steps run in containers unless NETDUCTOR_CI_HOST=1.
func Run(w *Workflow, workDir string, extraEnv []string) (log string, err error) {
	var b strings.Builder
	fmt.Fprintf(&b, "ci %s\n", ci.StatusLine())
	names := make([]string, 0, len(w.Jobs))
	for n := range w.Jobs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, jn := range names {
		job := w.Jobs[jn]
		img := containerImage(job.Container)
		if img == "" && ci.IsolationEnabled() {
			img = ci.DetectImage(workDir)
		}
		fmt.Fprintf(&b, "==> job %s (runs-on=%s container=%s)\n", jn, job.RunsOn, img)
		for i, st := range job.Steps {
			label := st.Name
			if label == "" {
				label = fmt.Sprintf("step-%d", i+1)
			}
			if st.Uses != "" && st.Run == "" {
				fmt.Fprintf(&b, "— skip uses: %s\n", st.Uses)
				continue
			}
			if strings.TrimSpace(st.Run) == "" {
				fmt.Fprintf(&b, "— skip empty step %s\n", label)
				continue
			}
			fmt.Fprintf(&b, "→ %s\n", label)
			dir := workDir
			if st.WorkingDirectory != "" {
				dir = filepath.Join(workDir, st.WorkingDirectory)
			}
			shell := st.Shell
			if shell == "" {
				shell = "bash"
			}
			env := append([]string{}, extraEnv...)
			for k, v := range job.Env {
				env = append(env, k+"="+v)
			}
			for k, v := range st.Env {
				env = append(env, k+"="+v)
			}
			out, e := ci.Exec(ci.ExecOpts{
				Image:   img,
				WorkDir: dir,
				Shell:   shell,
				Script:  st.Run,
				Env:     env,
			})
			b.WriteString(out)
			if len(out) > 0 && out[len(out)-1] != '\n' {
				b.WriteByte('\n')
			}
			if e != nil {
				fmt.Fprintf(&b, "FAIL %s: %v\n", label, e)
				return b.String(), fmt.Errorf("job %s step %s: %w", jn, label, e)
			}
		}
	}
	fmt.Fprintf(&b, "OK workflow %q\n", w.Name)
	return b.String(), nil
}

func FindDefault(root string) (string, error) {
	dir := filepath.Join(root, ".github", "workflows")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var files []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".yml") || strings.HasSuffix(n, ".yaml") {
			files = append(files, filepath.Join(dir, n))
		}
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no workflows in %s", dir)
	}
	sort.Strings(files)
	return files[0], nil
}
