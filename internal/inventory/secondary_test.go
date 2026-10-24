package inventory

import (
	"path/filepath"
	"testing"
)

func TestRegisterList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secondaries.json")
	s, err := Register(path, "peer-b", "B", "203.0.113.10", "test")
	if err != nil {
		t.Fatal(err)
	}
	if s.Lifecycle != "planned" {
		t.Fatal(s.Lifecycle)
	}
	items, err := List(path)
	if err != nil || len(items) != 1 {
		t.Fatalf("%v %#v", err, items)
	}
	if _, err := Register(path, "peer-b", "B2", "203.0.113.11", ""); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
