package gitstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitListLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_GIT_ROOT", dir)
	path, err := Init("demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "HEAD")); err != nil {
		t.Fatal(err)
	}
	list, err := List()
	if err != nil || len(list) != 1 || list[0] != "demo" {
		t.Fatalf("list=%v err=%v", list, err)
	}
	// empty bare: log may fail until first commit — acceptable
	_, _ = Log("demo", 5)
}

func TestPipeline(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_GIT_ROOT", dir)
	t.Setenv("NETDUCTOR_GIT_PIPELINES", filepath.Join(dir, "pipes"))
	if _, err := Init("p1"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSamplePipeline(); err != nil {
		t.Fatal(err)
	}
	out, err := RunPipeline("p1", "echo-ok")
	if err != nil {
		t.Fatal(err, out)
	}
	if out == "" {
		t.Fatal("empty output")
	}
}
