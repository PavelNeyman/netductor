package stack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	txPrepared      = "prepared"
	txNodeInstalled = "node_installed"
	txTGInstalled   = "tg_installed"
	txVerified      = "verified"
	txCommitted     = "committed"
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

func fileSHA(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func durableWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncDir(dir string) error {
	df, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = df.Sync()
	_ = df.Close()
	return err
}

func writeTx(att string, tx applyTx) error {
	b, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	return durableWrite(txPath(att), b, 0o600)
}

func loadTx(att string) (*applyTx, error) {
	b, err := os.ReadFile(txPath(att))
	if err != nil {
		return nil, err
	}
	var tx applyTx
	if err := json.Unmarshal(b, &tx); err != nil {
		return nil, fmt.Errorf("tx.json corrupt: %w", err)
	}
	if tx.Phase == "" || tx.NodeDest == "" || tx.TGDest == "" {
		return nil, fmt.Errorf("tx.json incomplete")
	}
	return &tx, nil
}

func durableBackup(src, dest string) (string, error) {
	b, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if err := durableWrite(dest, b, 0o755); err != nil {
		return "", err
	}
	return fileSHA(dest)
}

// RecoverIncompleteApply rolls back a pair install interrupted between Node and TG.
// Fail-closed: corrupt journal or missing pre-image blocks startup.
func RecoverIncompleteApply() error {
	return recoverApplyTx(attemptDir())
}

func recoverApplyTx(att string) error {
	tx, err := loadTx(att)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("apply recovery blocked: %w", err)
	}
	switch tx.Phase {
	case txCommitted:
		return nil
	case txPrepared, txNodeInstalled, txTGInstalled, txVerified:
		if err := rollbackPair(att, tx); err != nil {
			return fmt.Errorf("apply recovery blocked: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("apply recovery blocked: unknown phase %q", tx.Phase)
	}
}

func rollbackPair(att string, tx *applyTx) error {
	nodeBak := filepath.Join(att, "netductor")
	tgBak := filepath.Join(att, "netductor-tg")
	if tx.NodePreSHA != "" {
		if err := restoreChecked(nodeBak, tx.NodeDest, tx.NodePreSHA); err != nil {
			return err
		}
	}
	if tx.TGPreSHA != "" {
		if err := restoreChecked(tgBak, tx.TGDest, tx.TGPreSHA); err != nil {
			return err
		}
	}
	tx.Phase = txPrepared
	return writeTx(att, *tx)
}

func restoreChecked(src, dest, want string) error {
	got, err := fileSHA(src)
	if err != nil {
		return fmt.Errorf("pre-image %s: %w", src, err)
	}
	if got != want {
		return fmt.Errorf("pre-image %s sha mismatch", src)
	}
	if err := replaceExecutable(src, dest); err != nil {
		return err
	}
	after, err := fileSHA(dest)
	if err != nil {
		return err
	}
	if after != want {
		return fmt.Errorf("restored %s sha mismatch", dest)
	}
	return nil
}
