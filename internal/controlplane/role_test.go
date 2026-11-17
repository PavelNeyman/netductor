package controlplane

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultActive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	if Get() != RoleActive {
		t.Fatalf("default %q", Get())
	}
	if !WriteAllowed() {
		t.Fatal("writes should be allowed by default")
	}
}

func TestStandbyFencesWrites(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	if err := SetRole(RoleStandby); err != nil {
		t.Fatal(err)
	}
	if Get() != RoleStandby || WriteAllowed() {
		t.Fatalf("role=%s allowed=%v", Get(), WriteAllowed())
	}
	if err := SetRole(RoleActive); err != nil {
		t.Fatal(err)
	}
	if !WriteAllowed() {
		t.Fatal("active should allow writes")
	}
	b, _ := os.ReadFile(filepath.Join(dir, roleFileName))
	if string(b) != "active\n" {
		t.Fatalf("file %q", b)
	}
}
