package statefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	type row struct {
		N int `json:"n"`
	}
	if err := SaveJSON(path, row{N: 7}); err != nil {
		t.Fatal(err)
	}
	var got row
	if err := LoadJSON(path, &got); err != nil || got.N != 7 {
		t.Fatalf("load: %v %#v", err, got)
	}
}

func TestNotExist(t *testing.T) {
	var got struct{}
	err := LoadJSON(filepath.Join(t.TempDir(), "missing.json"), &got)
	if !IsNotExist(err) {
		t.Fatalf("want NotExist, got %v", err)
	}
}

func TestCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got struct{}
	if err := LoadJSON(path, &got); err == nil {
		t.Fatal("expected error")
	}
}
