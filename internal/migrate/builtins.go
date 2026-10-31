package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/aptguard"
	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/paths"
)

type ensureStateDirsMigration struct{}

func (m *ensureStateDirsMigration) ID() string { return "2026-10-ensure-state-dirs" }
func (m *ensureStateDirsMigration) Description() string {
	return "Ensure state/etc/migrate directory tree exists"
}
func (m *ensureStateDirsMigration) Required(ctx context.Context) (bool, error) {
	for _, d := range criticalDirs() {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			return true, nil
		}
	}
	// Secrets must not be world/group accessible
	sec := filepath.Join(paths.EtcDir(), "secrets")
	if st, err := os.Stat(sec); err == nil {
		if st.Mode().Perm()&0o077 != 0 {
			return true, nil
		}
	}
	return false, nil
}
func (m *ensureStateDirsMigration) Apply(ctx context.Context) error {
	for _, d := range criticalDirs() {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
		// Tighten sensitive trees (ignore errors on read-only FS)
		_ = os.Chmod(d, 0o700)
	}
	return nil
}

func criticalDirs() []string {
	return []string{
		paths.StateDir(),
		paths.EtcDir(),
		filepath.Join(paths.StateDir(), "migrate"),
		filepath.Join(paths.StateDir(), "stack"),
		filepath.Join(paths.StateDir(), "update", "locks"),
		filepath.Join(paths.StateDir(), "backups"),
		filepath.Join(paths.StateDir(), "edge"),
		filepath.Join(paths.StateDir(), "metrics"),
		filepath.Join(paths.StateDir(), "tg"),
		filepath.Join(paths.EtcDir(), "secrets"),
		filepath.Join(paths.EtcDir(), "secrets", "mtls"),
		filepath.Join(paths.EtcDir(), "sessions"),
		filepath.Join(paths.EtcDir(), "clients"),
	}
}
func (m *ensureStateDirsMigration) Verify(ctx context.Context) error {
	need, err := m.Required(ctx)
	if err != nil {
		return err
	}
	if need {
		return fmt.Errorf("state dirs still incomplete")
	}
	return nil
}

type repairStateOnUpgradeMigration struct{}

func (m *repairStateOnUpgradeMigration) ID() string {
	return "2026-10-repair-state-on-upgrade"
}
func (m *repairStateOnUpgradeMigration) Description() string {
	return "Idempotent LE hooks + state JSON mode repair (legacy RepairStateOnUpgrade)"
}
func repairMarker() string {
	return filepath.Join(paths.StateDir(), "migrate", "repair-state.done")
}
func (m *repairStateOnUpgradeMigration) Required(ctx context.Context) (bool, error) {
	if _, err := os.Stat(repairMarker()); err == nil {
		return false, nil
	}
	return true, nil
}
func (m *repairStateOnUpgradeMigration) Apply(ctx context.Context) error {
	_ = install.RepairStateOnUpgrade()
	_ = os.MkdirAll(filepath.Dir(repairMarker()), 0o700)
	return os.WriteFile(repairMarker(), []byte("ok\n"), 0o600)
}
func (m *repairStateOnUpgradeMigration) Verify(ctx context.Context) error {
	if _, err := os.Stat(repairMarker()); err != nil {
		return err
	}
	return nil
}

type ensureAptGuardPolicyMigration struct{}

func (m *ensureAptGuardPolicyMigration) ID() string {
	return "2026-10-ensure-aptguard-policy"
}
func (m *ensureAptGuardPolicyMigration) Description() string {
	return "Ensure APT Guard policy file exists (does not quarantine sources)"
}
func (m *ensureAptGuardPolicyMigration) Required(ctx context.Context) (bool, error) {
	_, err := os.Stat(aptguard.PolicyPath(paths.StateDir()))
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}
func (m *ensureAptGuardPolicyMigration) Apply(ctx context.Context) error {
	_, err := aptguard.EnsurePolicy(paths.StateDir())
	return err
}
func (m *ensureAptGuardPolicyMigration) Verify(ctx context.Context) error {
	need, err := m.Required(ctx)
	if err != nil {
		return err
	}
	if need {
		return fmt.Errorf("aptguard policy still missing")
	}
	return nil
}

type ensureRoleFileMigration struct{}

func (m *ensureRoleFileMigration) ID() string { return "2026-10-ensure-role-file" }
func (m *ensureRoleFileMigration) Description() string {
	return "Write /etc/netductor/role from DetectRole when absent"
}
func (m *ensureRoleFileMigration) Required(ctx context.Context) (bool, error) {
	path := filepath.Join(paths.EtcDir(), "role")
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}
func (m *ensureRoleFileMigration) Apply(ctx context.Context) error {
	role := firewall.DetectRole()
	if role == "" {
		role = "primary"
	}
	if err := os.MkdirAll(paths.EtcDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(paths.EtcDir(), "role"), []byte(role+"\n"), 0o644)
}
func (m *ensureRoleFileMigration) Verify(ctx context.Context) error {
	need, err := m.Required(ctx)
	if err != nil {
		return err
	}
	if need {
		return fmt.Errorf("role file still missing")
	}
	return nil
}
