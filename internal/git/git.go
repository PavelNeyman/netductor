package gitstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/gha"
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


func ArtifactDir() string {
	if v := os.Getenv("NETDUCTOR_GIT_ARTIFACTS"); v != "" {
		return v
	}
	return filepath.Join(paths.StateDir(), "git-artifacts")
}

func saveArtifact(repo, kind, name, body string) (string, error) {
	dir := filepath.Join(ArtifactDir(), sanitize(repo))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	fname := fmt.Sprintf("%s-%s-%s.log", ts, kind, sanitize(name))
	path := filepath.Join(dir, fname)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func ListArtifacts(repo string) ([]string, error) {
	dir := ArtifactDir()
	if repo != "" {
		dir = filepath.Join(dir, sanitize(repo))
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			if repo == "" {
				sub, _ := ListArtifacts(e.Name())
				out = append(out, sub...)
			}
			continue
		}
		if strings.HasSuffix(e.Name(), ".log") {
			if repo != "" {
				out = append(out, filepath.Join(sanitize(repo), e.Name()))
			} else {
				out = append(out, e.Name())
			}
		}
	}
	return out, nil
}

func ReadArtifact(rel string) (string, error) {
	rel = filepath.Clean(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") || strings.Contains(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid artifact path")
	}
	base := filepath.Clean(ArtifactDir())
	path := filepath.Join(base, rel)
	// contain under ArtifactDir
	sep := string(os.PathSeparator)
	if path != base && !strings.HasPrefix(path, base+sep) {
		return "", fmt.Errorf("artifact path escape")
	}
	b, err := os.ReadFile(path)
	return string(b), err
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
	// post-receive: optional auto pipeline via NETDUCTOR_GIT_PIPELINE
	hook := filepath.Join(dir, "hooks", "post-receive")
	hookBody := "#!/bin/sh\n" +
		"command -v netductor >/dev/null 2>&1 || exit 0\n" +
		"if [ -n \"$NETDUCTOR_GIT_WORKFLOW\" ]; then\n" +
		"  # 1 = default .github/workflows; else path relative to repo root\n" +
		"  if [ \"$NETDUCTOR_GIT_WORKFLOW\" = \"1\" ]; then\n" +
		"    netductor git workflow " + name + " || exit $?\n" +
		"  else\n" +
		"    netductor git workflow " + name + " \"$NETDUCTOR_GIT_WORKFLOW\" || exit $?\n" +
		"  fi\n" +
		"elif [ -n \"$NETDUCTOR_GIT_PIPELINE\" ]; then\n" +
		"  netductor git pipeline " + name + " \"$NETDUCTOR_GIT_PIPELINE\" || exit $?\n" +
		"fi\nexit 0\n"
	_ = os.WriteFile(hook, []byte(hookBody), 0o755)
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
	body := string(out)
	if path, e2 := saveArtifact(repo, "pipeline", pipeline, body); e2 == nil {
		body = body + "\n# artifact: " + path + "\n"
	}
	return body, err
}

func EnsureSamplePipeline() error {
	_ = os.MkdirAll(PipelineDir(), 0o755)
	samples := map[string]string{
		"echo-ok": `#!/bin/sh
# netductor-managed-pipeline
echo "pipeline ok repo=$REPO_NAME path=$REPO_PATH at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
`,
		"ci-run": `#!/bin/sh
# netductor-managed-pipeline — isolated container tests (no host toolchains)
set -e
WT=$(mktemp -d)
trap 'rm -rf "$WT"' EXIT
git --git-dir="$REPO_PATH" --work-tree="$WT" checkout -f HEAD 2>/dev/null || true
netductor ci test "$WT"
`,
		"go-test": `#!/bin/sh
# netductor-managed-pipeline — go test in golang container
set -e
WT=$(mktemp -d)
trap 'rm -rf "$WT"' EXIT
git --git-dir="$REPO_PATH" --work-tree="$WT" checkout -f HEAD 2>/dev/null || true
netductor ci exec --image "${NETDUCTOR_CI_IMAGE_GO:-golang:1.22-bookworm}" --workdir "$WT" -- "go test ./..."
`,
		"oci-push": `#!/bin/sh
# netductor-managed-pipeline — docker build on host engine, push to local registry
# (build runs via docker daemon; not language toolchains on host)
set -e
REG=${NETDUCTOR_REGISTRY_ADDR:-127.0.0.1:5000}
IMG=${OCI_IMAGE:-$REPO_NAME:latest}
WT=$(mktemp -d)
trap 'rm -rf "$WT"' EXIT
git --git-dir="$REPO_PATH" --work-tree="$WT" checkout -f HEAD 2>/dev/null || true
cd "$WT"
if [ ! -f Dockerfile ]; then
  echo "no Dockerfile — skip"
  netductor registry status || true
  exit 0
fi
if ! command -v docker >/dev/null 2>&1 && ! command -v podman >/dev/null 2>&1; then
  echo "docker/podman required for oci-push"
  exit 1
fi
ENG=$(command -v docker || command -v podman)
$ENG build -t "$REG/$IMG" .
if command -v crane >/dev/null 2>&1; then
  crane push "$REG/$IMG" "$REG/$IMG" 2>/dev/null || $ENG push "$REG/$IMG"
else
  $ENG push "$REG/$IMG"
fi
echo pushed "$REG/$IMG"
`,
	}
	for name, body := range samples {
		path := filepath.Join(PipelineDir(), name)
		if st, err := os.Stat(path); err == nil {
			b, _ := os.ReadFile(path)
			// refresh only managed or empty custom
			if !strings.Contains(string(b), "netductor-managed-pipeline") && st.Size() > 0 {
				continue
			}
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



// RunWorkflow checks out HEAD to a temp worktree and runs a GHA-subset YAML workflow.
// workflowPath relative to worktree or absolute; empty = first .github/workflows/*.yml
func RunWorkflow(repo, workflowPath string) (string, error) {
	dir, err := repoDir(repo)
	if err != nil {
		return "", err
	}
	wt, err := os.MkdirTemp("", "nd-wf-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(wt)
	cmd := exec.Command("git", "--git-dir="+dir, "--work-tree="+wt, "checkout", "-f", "HEAD")
	if out, err := cmd.CombinedOutput(); err != nil {
		return string(out), fmt.Errorf("checkout: %w: %s", err, strings.TrimSpace(string(out)))
	}
	path := workflowPath
	if path == "" {
		path, err = gha.FindDefault(wt)
		if err != nil {
			return "", err
		}
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(wt, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	w, err := gha.Parse(data)
	if err != nil {
		return "", err
	}
	extra := []string{
		"REPO_NAME=" + sanitize(repo),
		"REPO_PATH=" + dir,
		"GITHUB_WORKSPACE=" + wt,
		"CI=true",
	}
	log, err := gha.Run(w, wt, extra)
	wfName := filepath.Base(path)
	if ap, e2 := saveArtifact(repo, "workflow", wfName, log); e2 == nil {
		log = log + "\n# artifact: " + ap + "\n"
	}
	return log, err
}
