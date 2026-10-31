package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/aptguard"
	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/paths"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
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
	notes := install.RepairStateOnUpgrade()
	var hard []string
	for _, n := range notes {
		if strings.HasPrefix(n, "state bad:") || strings.HasPrefix(n, "le hooks:") || strings.HasPrefix(n, "chmod:") {
			hard = append(hard, n)
		}
	}
	if len(hard) > 0 {
		return fmt.Errorf("RepairStateOnUpgrade incomplete: %s", strings.Join(hard, "; "))
	}
	if err := os.MkdirAll(filepath.Dir(repairMarker()), 0o700); err != nil {
		return err
	}
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
	role := strings.TrimSpace(firewall.DetectRole())
	if role != "primary" && role != "secondary" {
		return fmt.Errorf("cannot write role file: DetectRole returned %q (need primary|secondary; refuse default primary)", role)
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

type ensureVPNDirsMigration struct{}

func (m *ensureVPNDirsMigration) ID() string { return "2026-10-ensure-vpn-dirs" }
func (m *ensureVPNDirsMigration) Description() string {
	return "Ensure VPN secrets/sessions/clients and empty vpn-users.json"
}
func (m *ensureVPNDirsMigration) Required(ctx context.Context) (bool, error) {
	users := filepath.Join(paths.EtcDir(), "vpn-users.json")
	if _, err := os.Stat(users); err != nil {
		return true, nil
	}
	for _, d := range []string{
		filepath.Join(paths.EtcDir(), "secrets"),
		filepath.Join(paths.EtcDir(), "sessions"),
		filepath.Join(paths.EtcDir(), "clients"),
	} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			return true, nil
		}
	}
	return false, nil
}
func (m *ensureVPNDirsMigration) Apply(ctx context.Context) error {
	return vpn.EnsureDirs()
}
func (m *ensureVPNDirsMigration) Verify(ctx context.Context) error {
	need, err := m.Required(ctx)
	if err != nil {
		return err
	}
	if need {
		return fmt.Errorf("vpn dirs still incomplete")
	}
	return nil
}

type ensureBaselineDirsMigration struct{}

func (m *ensureBaselineDirsMigration) ID() string { return "2026-10-ensure-baseline-dirs" }
func (m *ensureBaselineDirsMigration) Description() string {
	return "Ensure baseline/sites/git/ci state directories"
}
func (m *ensureBaselineDirsMigration) Required(ctx context.Context) (bool, error) {
	for _, d := range []string{
		filepath.Join(paths.StateDir(), "baseline"),
		filepath.Join(paths.StateDir(), "sites"),
		filepath.Join(paths.StateDir(), "git"),
		filepath.Join(paths.StateDir(), "ci"),
	} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			return true, nil
		}
	}
	return false, nil
}
func (m *ensureBaselineDirsMigration) Apply(ctx context.Context) error {
	for _, d := range []string{
		filepath.Join(paths.StateDir(), "baseline"),
		filepath.Join(paths.StateDir(), "sites"),
		filepath.Join(paths.StateDir(), "git"),
		filepath.Join(paths.StateDir(), "ci"),
		filepath.Join(paths.StateDir(), "alerts"),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}
func (m *ensureBaselineDirsMigration) Verify(ctx context.Context) error {
	need, err := m.Required(ctx)
	if err != nil {
		return err
	}
	if need {
		return fmt.Errorf("baseline/sites dirs still incomplete")
	}
	return nil
}

type ensureInstalledVersionMigration struct{}

func (m *ensureInstalledVersionMigration) ID() string { return "2026-10-ensure-installed-version" }
func (m *ensureInstalledVersionMigration) Description() string {
	return "Write state/installed_version from binary Release when missing or empty"
}
func (m *ensureInstalledVersionMigration) Required(ctx context.Context) (bool, error) {
	path := filepath.Join(paths.StateDir(), "installed_version")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return len(strings.TrimSpace(string(b))) == 0, nil
}
func (m *ensureInstalledVersionMigration) Apply(ctx context.Context) error {
	if err := os.MkdirAll(paths.StateDir(), 0o700); err != nil {
		return err
	}
	v := strings.TrimSpace(ndver.Release)
	if v == "" {
		v = "unknown"
	}
	return os.WriteFile(filepath.Join(paths.StateDir(), "installed_version"), []byte(v+"\n"), 0o644)
}
func (m *ensureInstalledVersionMigration) Verify(ctx context.Context) error {
	need, err := m.Required(ctx)
	if err != nil {
		return err
	}
	if need {
		return fmt.Errorf("installed_version still missing")
	}
	return nil
}
