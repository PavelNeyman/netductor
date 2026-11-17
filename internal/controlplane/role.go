// Package controlplane — Phase 3 active/standby control role (single writer).
package controlplane

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Role of this node for cluster source-of-truth.
type Role string

const (
	RoleActive  Role = "active"
	RoleStandby Role = "standby"
)

const roleFileName = "control_role"

var (
	mu       sync.Mutex
	testRole *Role // tests only
)

// RolePath is /var/lib/netductor/control_role (or NETDUCTOR_STATE_DIR).
func RolePath() string {
	return filepath.Join(paths.StateDir(), roleFileName)
}

// Get returns active if unset/empty (backward compatible single-control deploy).
func Get() Role {
	mu.Lock()
	defer mu.Unlock()
	if testRole != nil {
		return *testRole
	}
	b, err := os.ReadFile(RolePath())
	if err != nil {
		return RoleActive
	}
	s := strings.ToLower(strings.TrimSpace(string(b)))
	switch s {
	case string(RoleStandby), "passive", "secondary-control":
		return RoleStandby
	case string(RoleActive), "primary", "control", "":
		return RoleActive
	default:
		return RoleActive
	}
}

// WriteAllowed is false on standby — mutations must fail closed until promote.
func WriteAllowed() bool {
	return Get() == RoleActive
}

// SetRole writes role file. Use RoleActive only after intentional promote.
func SetRole(r Role) error {
	switch r {
	case RoleActive, RoleStandby:
	default:
		return fmt.Errorf("control role must be active or standby, got %q", r)
	}
	mu.Lock()
	defer mu.Unlock()
	if testRole != nil {
		*testRole = r
		return nil
	}
	dir := paths.StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, roleFileName)
	tmp := path + ".tmp"
	payload := []byte(string(r) + "\n")
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// SetTestRole overrides role for unit tests (nil clears).
func SetTestRole(r *Role) {
	mu.Lock()
	defer mu.Unlock()
	testRole = r
}
