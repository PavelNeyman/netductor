package stack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/install"
)

const (
	txPrepared      = "prepared"
	txNodeInstalled = "node_installed"
	txTGInstalled   = "tg_installed"
	txVerified      = "verified"
	txCommitted     = "committed"
	txRolledBack    = "rolled_back"
)

type applyTx struct {
	ID         string    `json:"id"`
	NodeDest   string    `json:"node_dest"`
	TGDest     string    `json:"tg_dest"`
	NodePreSHA string    `json:"node_pre_sha"`
	TGPreSHA   string    `json:"tg_pre_sha"`
	NodeNewSHA string    `json:"node_new_sha"`
	TGNewSHA   string    `json:"tg_new_sha"`
	Phase      string    `json:"phase"`
	StartedAt  time.Time `json:"started_at"`
}

func txPath(att string) string { return filepath.Join(att, "tx.json") }

func validSHA(s string) bool {
	if len(s) != sha256.Size*2 { return false }
	_, err := hex.DecodeString(s)
	return err == nil
}

func fileSHA(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil { return "", err }
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func durableWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil { return err }
	if _, err = f.Write(data); err != nil { _ = f.Close(); _ = os.Remove(tmp); return err }
	if err = f.Sync(); err != nil { _ = f.Close(); _ = os.Remove(tmp); return err }
	if err = f.Close(); err != nil { _ = os.Remove(tmp); return err }
	if err = os.Rename(tmp, path); err != nil { _ = os.Remove(tmp); return err }
	return syncDir(filepath.Dir(path))
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil { return err }
	syncErr := f.Sync()
	closeErr := f.Close()
	if syncErr != nil { return syncErr }
	return closeErr
}

func writeTx(att string, tx applyTx) error {
	b, err := json.MarshalIndent(tx, "", "  ")
	if err != nil { return err }
	return durableWrite(txPath(att), append(b, '\n'), 0o600)
}

func validateTx(tx *applyTx) error {
	if tx == nil || strings.TrimSpace(tx.ID) == "" || tx.StartedAt.IsZero() {
		return fmt.Errorf("tx.json missing identity or start time")
	}
	if !filepath.IsAbs(tx.NodeDest) || !filepath.IsAbs(tx.TGDest) ||
		filepath.Clean(tx.NodeDest) == filepath.Clean(tx.TGDest) {
		return fmt.Errorf("tx.json has invalid destination paths")
	}
	switch tx.Phase {
	case txPrepared, txNodeInstalled, txTGInstalled, txVerified, txCommitted, txRolledBack:
	default:
		return fmt.Errorf("tx.json has unknown phase %q", tx.Phase)
	}
	for name, value := range map[string]string{
		"node_pre_sha": tx.NodePreSHA, "tg_pre_sha": tx.TGPreSHA,
		"node_new_sha": tx.NodeNewSHA, "tg_new_sha": tx.TGNewSHA,
	} {
		if !validSHA(value) { return fmt.Errorf("tx.json has invalid %s", name) }
	}
	return nil
}

func loadTx(att string) (*applyTx, error) {
	b, err := os.ReadFile(txPath(att))
	if err != nil { return nil, err }
	var tx applyTx
	if err := json.Unmarshal(b, &tx); err != nil { return nil, fmt.Errorf("tx.json corrupt: %w", err) }
	if err := validateTx(&tx); err != nil { return nil, err }
	return &tx, nil
}

func durableBackup(src, dest string) (string, error) {
	b, err := os.ReadFile(src)
	if err != nil { return "", err }
	if err := durableWrite(dest, b, 0o755); err != nil { return "", err }
	return fileSHA(dest)
}

func verifyInstalledPair(tx *applyTx) error {
	for _, item := range []struct{name, path, want string}{
		{"node", tx.NodeDest, tx.NodeNewSHA}, {"tg", tx.TGDest, tx.TGNewSHA},
	} {
		got, err := fileSHA(item.path)
		if err != nil { return fmt.Errorf("verify installed %s: %w", item.name, err) }
		if got != item.want { return fmt.Errorf("verify installed %s: sha256 mismatch", item.name) }
	}
	return nil
}

func verifyPreImagePair(att string, tx *applyTx) error {
	for _, item := range []struct{name, path, want string}{
		{"node", filepath.Join(att, "netductor"), tx.NodePreSHA},
		{"tg", filepath.Join(att, "netductor-tg"), tx.TGPreSHA},
	} {
		got, err := fileSHA(item.path)
		if err != nil { return fmt.Errorf("verify %s pre-image: %w", item.name, err) }
		if got != item.want { return fmt.Errorf("verify %s pre-image: sha256 mismatch", item.name) }
	}
	return nil
}

// commitInstalledPair is called only after service health, host baseline and
// last-good snapshot preparation have succeeded.
func commitInstalledPair(att string) error {
	tx, err := loadTx(att)
	if err != nil { return fmt.Errorf("load apply transaction: %w", err) }
	if tx.Phase != txTGInstalled && tx.Phase != txVerified {
		return fmt.Errorf("cannot commit apply transaction in phase %q", tx.Phase)
	}
	if err := verifyInstalledPair(tx); err != nil { return err }
	tx.Phase = txVerified
	if err := writeTx(att, *tx); err != nil { return fmt.Errorf("persist verified phase: %w", err) }
	tx.Phase = txCommitted
	if err := writeTx(att, *tx); err != nil { return fmt.Errorf("persist committed phase: %w", err) }
	return nil
}

// Apply owns the lock while restarting the API. During that bounded interval
// startup must allow the installed pair to boot so Apply can perform health checks.
func applyOwnerActive() bool {
	if applyUnitRunning() { return true }
	b, err := os.ReadFile(applyLockFile)
	if err != nil { return false }
	parts := strings.SplitN(strings.TrimSpace(string(b)), "\t", 3)
	if len(parts) < 2 { return false }
	started, err := time.Parse(time.RFC3339, parts[0])
	if err != nil || time.Since(started) < 0 { return false }
	if len(parts) == 3 { return applyPIDRunning(parts[2]) }
	return time.Since(started) < applyLockTTL
}

func recoverIncompleteApply() (bool, error) {
	if err := recoverPrevSnapshotSwap(); err != nil {
		return false, fmt.Errorf("last-good snapshot recovery blocked: %w", err)
	}
	tx, err := loadTx(attemptDir())
	if err != nil {
		if os.IsNotExist(err) { return false, nil }
		return false, fmt.Errorf("apply recovery blocked: %w", err)
	}
	willRollback := false
	switch tx.Phase {
	case txPrepared, txNodeInstalled, txTGInstalled, txVerified:
		willRollback = !applyOwnerActive()
	}
	if err := recoverApplyTx(attemptDir()); err != nil { return false, err }
	if willRollback {
		if err := install.EnsureHostBaseline(firewall.DetectRole()); err != nil {
			return true, fmt.Errorf("apply pair rolled back but host baseline repair failed: %w", err)
		}
	}
	return willRollback, nil
}

// RecoverIncompleteApply is used by non-API callers. If recovery rolled back
// the pair, restart both services so their mapped executable matches disk.
func RecoverIncompleteApply() error {
	rolledBack, err := recoverIncompleteApply()
	if err != nil { return err }
	if !rolledBack { return nil }
	var restartErrs []string
	for _, unit := range []string{"netductor-api", "netductor-telegram-bot"} {
		if err := exec.Command("systemctl", "restart", unit).Run(); err != nil { restartErrs = append(restartErrs, unit+": "+err.Error()) }
	}
	if len(restartErrs) > 0 { return fmt.Errorf("apply pair rolled back; service restart failed: %s", strings.Join(restartErrs, "; ")) }
	if !waitAPIHealthy(20*time.Second) { return fmt.Errorf("apply pair rolled back; API health check failed") }
	return nil
}

// RecoverIncompleteApplyForStartup reports whether startup recovery replaced
// the on-disk pair. The caller must re-exec itself before opening listeners.
func RecoverIncompleteApplyForStartup() (bool, error) {
	return recoverIncompleteApply()
}

func recoverApplyTx(att string) error {
	tx, err := loadTx(att)
	if err != nil {
		if os.IsNotExist(err) { return nil }
		return fmt.Errorf("apply recovery blocked: %w", err)
	}
	switch tx.Phase {
	case txCommitted:
		if err := verifyInstalledPair(tx); err != nil { return fmt.Errorf("committed apply recovery blocked: %w", err) }
		return nil
	case txRolledBack:
		if err := verifyPreImagePair(att, tx); err != nil { return fmt.Errorf("rolled-back apply recovery blocked: %w", err) }
		return nil
	case txTGInstalled, txVerified:
		if applyOwnerActive() {
			if err := verifyInstalledPair(tx); err != nil { return fmt.Errorf("active apply pair verification failed: %w", err) }
			return nil
		}
		if err := rollbackPair(att, tx); err != nil { return fmt.Errorf("apply recovery blocked: %w", err) }
		return nil
	case txPrepared, txNodeInstalled:
		if applyOwnerActive() {
			return fmt.Errorf("apply recovery blocked: transaction %s is still mutating the binary pair", tx.ID)
		}
		if err := rollbackPair(att, tx); err != nil { return fmt.Errorf("apply recovery blocked: %w", err) }
		return nil
	default:
		return fmt.Errorf("apply recovery blocked: unknown phase %q", tx.Phase)
	}
}

func rollbackPair(att string, tx *applyTx) error {
	if err := verifyPreImagePair(att, tx); err != nil { return err }
	if err := restoreChecked(filepath.Join(att, "netductor"), tx.NodeDest, tx.NodePreSHA); err != nil {
		return fmt.Errorf("restore node: %w", err)
	}
	if err := restoreChecked(filepath.Join(att, "netductor-tg"), tx.TGDest, tx.TGPreSHA); err != nil {
		return fmt.Errorf("restore tg: %w", err)
	}
	tx.Phase = txRolledBack
	return writeTx(att, *tx)
}

func restoreChecked(src, dest, want string) error {
	if !validSHA(want) { return fmt.Errorf("invalid expected pre-image hash for %s", dest) }
	got, err := fileSHA(src)
	if err != nil { return fmt.Errorf("pre-image %s: %w", src, err) }
	if got != want { return fmt.Errorf("pre-image %s sha mismatch", src) }
	if err := replaceExecutable(src, dest); err != nil { return err }
	after, err := fileSHA(dest)
	if err != nil { return err }
	if after != want { return fmt.Errorf("restored %s sha mismatch", dest) }
	return nil
}

func cleanupTerminalAttempt(att string) error {
	tx, err := loadTx(att)
	if err != nil {
		if os.IsNotExist(err) { return os.RemoveAll(att) }
		return err
	}
	if tx.Phase != txCommitted && tx.Phase != txRolledBack {
		return fmt.Errorf("refuse to remove non-terminal apply transaction %s (%s)", tx.ID, tx.Phase)
	}
	return os.RemoveAll(att)
}
