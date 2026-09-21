package gitstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func Root() string {
	if v := os.Getenv("NETDUCTOR_GIT_ROOT"); v != "" {
		return v
	}
	return filepath.Join(paths.StateDir(), "git")
}

func PipelineDir() string {
	if v := os.Getenv("NETDUCTOR_GIT_PIPELINES"); v != "" {
		return v
	}
	return filepath.Join(paths.EtcDir(), "git-pipelines")
}

func EnsureRoot() error {
	return os.MkdirAll(Root(), 0o700)
}

func List() ([]string, error) {
	if err := EnsureRoot(); err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(Root())
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".git") {
			if _, err := os.Stat(filepath.Join(Root(), name, "HEAD")); err == nil {
				out = append(out, strings.TrimSuffix(name, ".git"))
			}
		}
	}
	return out, nil
}

func Init(name string) (string, error) {
	name = sanitize(name)
	if name == "" {
		return "", fmt.Errorf("empty name")
	}
	if err := EnsureRoot(); err != nil {
		return "", err
	}
	dir := filepath.Join(Root(), name+".git")
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err == nil {
		return dir, fmt.Errorf("already exists: %s", name)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "init", "--bare", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	// sample post-receive that can call netductor pipeline
	hook := filepath.Join(dir, "hooks", "post-receive")
	_ = os.WriteFile(hook, []byte("#!/bin/sh\n# Optional: netductor git pipeline "+name+" default\nexit 0\n"), 0o755)
	return dir, nil
}

func Delete(name string) error {
	name = sanitize(name)
	if name == "" {
		return fmt.Errorf("empty name")
	}
	dir := filepath.Join(Root(), name+".git")
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		return fmt.Errorf("repo not found: %s", name)
	}
	return os.RemoveAll(dir)
}

func Log(name string, n int) (string, error) {
	dir, err := repoDir(name)
	if err != nil {
		return "", err
	}
	if n <= 0 {
		n = 20
	}
	cmd := exec.Command("git", "-C", dir, "log", fmt.Sprintf("-%d", n), "--format=%h%x09%s%x09%an%x09%ai")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func Show(name, rev string) (string, error) {
	dir, err := repoDir(name)
	if err != nil {
		return "", err
	}
	if rev == "" {
		rev = "HEAD"
	}
	cmd := exec.Command("git", "-C", dir, "show", "--stat", "-p", "--format=fuller", rev)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ListPipelines returns script names in PipelineDir (executable or .sh).
func ListPipelines() ([]string, error) {
	_ = os.MkdirAll(PipelineDir(), 0o755)
	ents, err := os.ReadDir(PipelineDir())
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// RunPipeline executes PipelineDir/<pipeline> with env REPO_NAME, REPO_PATH, optional args.
func RunPipeline(repo, pipeline string, extraArgs ...string) (string, error) {
	dir, err := repoDir(repo)
	if err != nil {
		return "", err
	}
	pipeline = filepath.Base(pipeline)
	script := filepath.Join(PipelineDir(), pipeline)
	st, err := os.Stat(script)
	if err != nil {
		return "", fmt.Errorf("pipeline not found: %s (put scripts in %s)", pipeline, PipelineDir())
	}
	if st.Mode()&0o111 == 0 {
		_ = os.Chmod(script, st.Mode()|0o755)
	}
	args := append([]string{script}, extraArgs...)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"REPO_NAME="+sanitize(repo),
		"REPO_PATH="+dir,
		"NETDUCTOR_GIT_ROOT="+Root(),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func EnsureSamplePipeline() error {
	_ = os.MkdirAll(PipelineDir(), 0o755)
	samples := map[string]string{
		"echo-ok": "#!/bin/sh\necho \"pipeline ok repo=$REPO_NAME path=$REPO_PATH at $(date -u +%Y-%m-%dT%H:%M:%SZ)\"\n",
		"go-test": "#!/bin/sh\nset -e\necho \"go-test $REPO_NAME\"\n# clone bare to worktree\nWT=$(mktemp -d)\ngit --git-dir=\"$REPO_PATH\" --work-tree=\"$WT\" checkout -f HEAD 2>/dev/null || true\ncd \"$WT\"\nif [ -f go.mod ]; then go test ./...; else echo no go.mod; fi\nrm -rf \"$WT\"\n",
	}
	for name, body := range samples {
		path := filepath.Join(PipelineDir(), name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func RemoteHint(name string, sshPort int) string {
	name = sanitize(name)
	if sshPort <= 0 {
		sshPort = 52222
	}
	return fmt.Sprintf("ssh://root@HOST:%d%s/%s.git", sshPort, Root(), name)
}

func repoDir(name string) (string, error) {
	name = sanitize(name)
	dir := filepath.Join(Root(), name+".git")
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		return "", fmt.Errorf("repo not found: %s", name)
	}
	return dir, nil
}

func sanitize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".git")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Info is for JSON APIs.
type Info struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ListInfo() ([]Info, error) {
	names, err := List()
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, n := range names {
		out = append(out, Info{Name: n, Path: filepath.Join(Root(), n+".git")})
	}
	return out, nil
}

