package gha

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectWorkflow(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	must := func(dir, name, body string) {
		_ = os.MkdirAll(dir, 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(dir, "ci.yml", "name: ci\njobs:\n  t:\n    steps:\n      - run: echo x\n")
	must(dir, "larvatus.yml", "name: larvatus\njobs:\n  t:\n    steps:\n      - run: echo y\n")
	path, err := ResolveProjectWorkflow(root, "larvatus")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "larvatus.yml" {
		t.Fatalf("got %s", path)
	}
	if _, err := ResolveProjectWorkflow(root, "missing"); err == nil {
		t.Fatal("expected error")
	}
	// ci fallback
	root2 := t.TempDir()
	must(filepath.Join(root2, "ci"), "paenn.yml", "name: paenn\njobs:\n  t:\n    steps:\n      - run: echo z\n")
	path, err = ResolveProjectWorkflow(root2, "paenn")
	if err != nil || filepath.Base(path) != "paenn.yml" {
		t.Fatalf("ci fallback %v %s", err, path)
	}
}
