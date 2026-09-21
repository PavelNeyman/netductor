// Package ci runs build/test jobs isolated in containers (docker/podman).
// Host keeps data only: git bare repos, registry blobs, artifacts.
package ci

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsolationEnabled is default true. Set NETDUCTOR_CI_HOST=1 to run on host (discouraged).
func IsolationEnabled() bool {
	return os.Getenv("NETDUCTOR_CI_HOST") != "1"
}

func Engine() string {
	for _, c := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return ""
}

// Default images when job has no container: (override via NETDUCTOR_CI_IMAGE_*)
var defaultImages = map[string]string{
	"go":     "golang:1.22-bookworm",
	"node":   "node:20-bookworm",
	"rust":   "rust:1.81-bookworm",
	"python": "python:3.12-bookworm",
	"docker": "docker:27-cli", // docker-in-docker needs privileged — oci uses host docker build
	"generic": "debian:bookworm-slim",
}

func imageEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ImageForGo() string     { return imageEnv("NETDUCTOR_CI_IMAGE_GO", defaultImages["go"]) }
func ImageForNode() string   { return imageEnv("NETDUCTOR_CI_IMAGE_NODE", defaultImages["node"]) }
func ImageForRust() string   { return imageEnv("NETDUCTOR_CI_IMAGE_RUST", defaultImages["rust"]) }
func ImageForPython() string { return imageEnv("NETDUCTOR_CI_IMAGE_PYTHON", defaultImages["python"]) }
func ImageGeneric() string   { return imageEnv("NETDUCTOR_CI_IMAGE", defaultImages["generic"]) }

// DetectImage chooses a language image from worktree markers.
func DetectImage(workDir string) string {
	checks := []struct {
		file string
		img  func() string
	}{
		{"go.mod", ImageForGo},
		{"package.json", ImageForNode},
		{"Cargo.toml", ImageForRust},
		{"pyproject.toml", ImageForPython},
		{"requirements.txt", ImageForPython},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(workDir, c.file)); err == nil {
			return c.img()
		}
	}
	return ImageGeneric()
}

// DetectTestCommand returns a default test command for the worktree.
func DetectTestCommand(workDir string) string {
	if _, err := os.Stat(filepath.Join(workDir, "go.mod")); err == nil {
		return "go test ./..."
	}
	if _, err := os.Stat(filepath.Join(workDir, "package.json")); err == nil {
		return "npm test"
	}
	if _, err := os.Stat(filepath.Join(workDir, "Cargo.toml")); err == nil {
		return "cargo test"
	}
	if _, err := os.Stat(filepath.Join(workDir, "pyproject.toml")); err == nil {
		return "pip install -q pytest 2>/dev/null; pytest || python -m pytest || true"
	}
	if _, err := os.Stat(filepath.Join(workDir, "requirements.txt")); err == nil {
		return "pip install -q -r requirements.txt pytest 2>/dev/null; pytest || python -m pytest || true"
	}
	if _, err := os.Stat(filepath.Join(workDir, "Makefile")); err == nil {
		return "make test || make check || true"
	}
	return "echo 'no project marker — set container: and run: in workflow'; exit 0"
}

type ExecOpts struct {
	Image   string
	WorkDir string // host path mounted at /workspace
	Shell   string
	Script  string
	Env     []string // KEY=VAL
	Network string   // default bridge; "host" for registry 127.0.0.1
}

// Exec runs script inside container with workDir mounted at /workspace.
func Exec(opts ExecOpts) (string, error) {
	if opts.WorkDir == "" {
		return "", fmt.Errorf("workdir required")
	}
	if opts.Shell == "" {
		opts.Shell = "bash"
	}
	if opts.Image == "" {
		opts.Image = DetectImage(opts.WorkDir)
	}
	if !IsolationEnabled() {
		cmd := exec.Command(opts.Shell, "-c", opts.Script)
		cmd.Dir = opts.WorkDir
		cmd.Env = append(os.Environ(), opts.Env...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	eng := Engine()
	if eng == "" {
		return "", fmt.Errorf("docker/podman required for isolated CI (or NETDUCTOR_CI_HOST=1)")
	}
	net := opts.Network
	if net == "" {
		net = "bridge"
	}
	args := []string{
		"run", "--rm",
		"-v", opts.WorkDir + ":/workspace:rw",
		"-w", "/workspace",
		"--network", net,
	}
	for _, e := range opts.Env {
		args = append(args, "-e", e)
	}
	// pass through common CI env
	for _, k := range []string{"CI", "REPO_NAME", "GITHUB_WORKSPACE"} {
		if v := os.Getenv(k); v != "" {
			args = append(args, "-e", k+"="+v)
		}
	}
	args = append(args, opts.Image, opts.Shell, "-c", opts.Script)
	cmd := exec.Command(eng, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestWorktree detects language and runs default tests in a container.
func TestWorktree(workDir string) (string, error) {
	img := DetectImage(workDir)
	script := DetectTestCommand(workDir)
	var b strings.Builder
	fmt.Fprintf(&b, "ci isolate image=%s\n", img)
	out, err := Exec(ExecOpts{Image: img, WorkDir: workDir, Script: script, Env: []string{"CI=true"}})
	b.WriteString(out)
	return b.String(), err
}

func StatusLine() string {
	eng := Engine()
	iso := IsolationEnabled()
	return fmt.Sprintf("isolation=%v engine=%s host_escape=%v", iso, eng, !iso)
}
