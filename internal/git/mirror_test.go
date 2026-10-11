package gitstore

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSetBareHEADMainNotMaster(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "t.git")
	if out, err := exec.Command("git", "init", "--bare", repo).CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, out)
	}
	// seed a non-bare with main only, push into bare
	src := filepath.Join(dir, "src")
	_ = os.MkdirAll(src, 0o755)
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v %v: %s", args, err, out)
		}
	}
	run(src, "init", "-b", "main")
	run(src, "config", "user.email", "t@t")
	run(src, "config", "user.name", "t")
	_ = os.WriteFile(filepath.Join(src, "a"), []byte("x"), 0o644)
	run(src, "add", "a")
	run(src, "commit", "-m", "c")
	run(src, "remote", "add", "origin", repo)
	run(src, "push", "origin", "main")
	// HEAD still default master on bare — broken
	if err := setBareHEAD(repo); err != nil {
		t.Fatal(err)
	}
	if !bareHasResolvableHEAD(repo) {
		t.Fatal("HEAD still unresolved")
	}
	out, _ := exec.Command("git", "-C", repo, "symbolic-ref", "--short", "HEAD").CombinedOutput()
	if string(out) != "main\n" && string(out) != "main" {
		t.Fatalf("want main got %q", out)
	}
}
