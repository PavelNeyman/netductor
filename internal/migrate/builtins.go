package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/aptguard"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/paths"
)

type ensureStateDirsMigration struct{}

func (m *ensureStateDirsMigration) ID() string { return "2026-10-ensure-state-dirs" }
func (m *ensureStateDirsMigration) Description() string {
	return "Ensure state/etc/migrate directory tree exists"
}
func (m *ensureStateDirsMigration) Required(ctx context.Context) (bool, error) {
	for _, d := range []string{
		paths.StateDir(),
		paths.EtcDir(),
		filepath.Join(paths.StateDir(), "migrate"),
		filepath.Join(paths.StateDir(), "stack"),
		filepath.Join(paths.StateDir(), "update", "locks"),
	} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			return true, nil
		}
	}
	return false, nil
}
func (m *ensureStateDirsMigration) Apply(ctx context.Context) error {
	for _, d := range []string{
		paths.StateDir(),
		paths.EtcDir(),
		filepath.Join(paths.StateDir(), "migrate"),
		filepath.Join(paths.StateDir(), "stack"),
		filepath.Join(paths.StateDir(), "update", "locks"),
		filepath.Join(paths.StateDir(), "backups"),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
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
