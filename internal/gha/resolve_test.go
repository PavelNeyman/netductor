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
	must := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("ci.yml", "name: ci\njobs:\n  t:\n    steps:\n      - run: echo x\n")
	must("larvatus.yml", "name: larvatus\njobs:\n  t:\n    steps:\n      - run: echo y\n")
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
	// ci/ must NOT resolve
	root2 := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root2, "ci"), 0o755)
	_ = os.WriteFile(filepath.Join(root2, "ci", "paenn.yml"), []byte("name: paenn\njobs:\n  t:\n    steps:\n      - run: echo z\n"), 0o644)
	if _, err := ResolveProjectWorkflow(root2, "paenn"); err == nil {
		t.Fatal("ci/ fallback must not resolve")
	}
}
