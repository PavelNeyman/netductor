package stack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerLess(t *testing.T) {
	if !verLess("0.9.121", "0.9.133") {
		t.Fatal("121 < 133")
	}
	if verLess("0.9.133", "0.9.121") {
		t.Fatal("133 not < 121")
	}
	if verLess("v0.9.170", "0.9.170") {
		t.Fatal("equal after norm")
	}
	if !verLess("0.9.9", "0.9.10") {
		t.Fatal("9 < 10")
	}
}

func TestPinUnpin(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	if ok, _ := IsPinned(); ok {
		t.Fatal("should not be pinned")
	}
	if err := Pin("test pin"); err != nil {
		t.Fatal(err)
	}
	ok, why := IsPinned()
	if !ok || why == "" {
		t.Fatalf("pinned=%v why=%q", ok, why)
	}
	if _, err := os.Stat(filepath.Join(dir, "stack", "PIN")); err != nil {
		t.Fatal(err)
	}
	_ = Unpin()
	if ok, _ := IsPinned(); ok {
		t.Fatal("still pinned")
	}
}

func TestSaveCurrentFromInstalledNodeUsesOnDiskBinary(t *testing.T) {
	state := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", state)
	bin := filepath.Join(t.TempDir(), "netductor")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'netductor 0.9.612 (node)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	output := binVersion(bin)
	release := verNorm(parseVerField(output))
	if release != "0.9.612" {
		t.Fatalf("release=%q output=%q", release, output)
	}
	saveCurrent(release)
	b, err := os.ReadFile(filepath.Join(state, "stack", "current.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"release": "0.9.612"`) {
		t.Fatalf("current metadata does not reflect installed binary: %s", b)
	}
}

func TestPrimaryUnitsRoles(t *testing.T) {
	if len(PrimaryUnits) < 4 {
		t.Fatal(PrimaryUnits)
	}
	seen := map[string]bool{}
	for _, u := range PrimaryUnits {
		if u.Unit == "" || u.Role == "" {
			t.Fatalf("%+v", u)
		}
		if seen[u.Unit] {
			t.Fatalf("dup unit %s", u.Unit)
		}
		seen[u.Unit] = true
	}
}
