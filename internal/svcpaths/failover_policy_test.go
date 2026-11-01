package svcpaths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSavePolicyFileAtomicAndChecked(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chown(dir, os.Geteuid(), os.Getegid()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "policy.json")
	want := Policy{
		Enabled:              true,
		UsersToSPOnVLESSDown: true,
		FailThreshold:        4,
		OKThreshold:          5,
		StableSeconds:        60,
	}
	if err := savePolicyFile(path, want); err != nil {
		t.Fatalf("savePolicyFile() error = %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Policy
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("saved policy is invalid JSON: %v", err)
	}
	if got != want {
		t.Fatalf("saved policy = %+v, want %+v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("policy mode = %o, want 644", info.Mode().Perm())
	}

	link := filepath.Join(dir, "policy-link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := savePolicyFile(link, DefaultPolicy()); err == nil {
		t.Fatal("savePolicyFile() accepted a symlink target")
	}
	bAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(bAfter) != string(b) {
		t.Fatal("symlink rejection changed the original policy")
	}
}
