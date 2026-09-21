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
	return dir, nil
}

func Log(name string, n int) (string, error) {
	dir, err := repoDir(name)
	if err != nil {
		return "", err
	}
	if n <= 0 {
		n = 20
	}
	cmd := exec.Command("git", "-C", dir, "log", fmt.Sprintf("-%d", n), "--oneline", "--decorate")
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
	cmd := exec.Command("git", "-C", dir, "show", "--stat", rev)
	out, err := cmd.CombinedOutput()
	return string(out), err
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
