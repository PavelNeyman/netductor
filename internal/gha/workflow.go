// Package gha runs a minimal GitHub Actions–compatible subset:
// jobs.*.steps with `run:` (shell). Ignores uses:/services/matrix.
package gha

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name string             `yaml:"name"`
	On   any                `yaml:"on"`
	Jobs map[string]Job     `yaml:"jobs"`
}

type Job struct {
	RunsOn string  `yaml:"runs-on"`
	Steps  []Step  `yaml:"steps"`
	Env    map[string]string `yaml:"env"`
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

// Parse loads GHA-like YAML.
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

type Result struct {
	Job   string
	Step  string
	Output string
	Err   error
}

// Run executes jobs in sorted name order; only steps with non-empty run:.
// workDir is the checkout / work tree root.
func Run(w *Workflow, workDir string, extraEnv []string) (log string, err error) {
	var b strings.Builder
	names := make([]string, 0, len(w.Jobs))
	for n := range w.Jobs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, jn := range names {
		job := w.Jobs[jn]
		fmt.Fprintf(&b, "==> job %s (runs-on=%s)\n", jn, job.RunsOn)
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
			cmd := exec.Command(shell, "-c", st.Run)
			cmd.Dir = dir
			env := append(os.Environ(), extraEnv...)
			for k, v := range job.Env {
				env = append(env, k+"="+v)
			}
			for k, v := range st.Env {
				env = append(env, k+"="+v)
			}
			cmd.Env = env
			out, e := cmd.CombinedOutput()
			b.Write(out)
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

// FindDefault looks for .github/workflows/*.yml under root (first name sorted).
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
