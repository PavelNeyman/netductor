package mtls

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/statefile"
	"time"
)

const pendingRotateFile = "pending_rotate.json"

type pendingRotate struct {
	NodeID    string `json:"node_id"`
	OldSerial string `json:"old_serial"`
	NewSerial string `json:"new_serial"`
	ExpiresAt int64  `json:"expires_at"` // unix; after this force-revoke old
	CreatedAt int64  `json:"created_at"`
}

var pendingMu sync.Mutex

func pendingPath() string { return filepath.Join(Dir(), pendingRotateFile) }

func loadPending() (map[string]pendingRotate, error) {
	out := map[string]pendingRotate{}
	var list []pendingRotate
	err := statefile.LoadJSON(pendingPath(), &list)
	if statefile.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	for _, p := range list {
		if strings.TrimSpace(p.NodeID) == "" || strings.TrimSpace(p.NewSerial) == "" {
			return nil, fmt.Errorf("invalid pending rotate entry")
		}
		if p.ExpiresAt > 0 && now > p.ExpiresAt {
			if p.OldSerial != "" {
				if err := RevokeSerial(p.OldSerial, p.NodeID, "grace-expired"); err != nil {
					return nil, fmt.Errorf("revoke expired client certificate for %s: %w", p.NodeID, err)
				}
			}
			continue
		}
		if _, exists := out[p.NodeID]; exists {
			return nil, fmt.Errorf("duplicate pending rotate entry for %s", p.NodeID)
		}
		out[p.NodeID] = p
	}
	return out, nil
}

func savePending(m map[string]pendingRotate) error {
	list := make([]pendingRotate, 0, len(m))
	for _, p := range m {
		list = append(list, p)
	}
	return statefile.SaveJSON(pendingPath(), list)
}

// GraceHours after rotate before old serial is force-revoked (default 24).
func GraceHours() int {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_MTLS_GRACE_HOURS")); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return 24
}

// RotateClientForPush re-issues cert, keeping the previous certificate valid until
// Confirm or grace expiry. Existing pending rotations are returned idempotently.
func recoverRotationBackup(nodeID string, pending map[string]pendingRotate) error {
	dir := ClientDir(nodeID)
	certPath := filepath.Join(dir, ClientCertFile)
	keyPath := filepath.Join(dir, ClientKeyFile)
	certBackup := certPath + ".rotate-backup"
	keyBackup := keyPath + ".rotate-backup"

	_, certBackupErr := os.Lstat(certBackup)
	_, keyBackupErr := os.Lstat(keyBackup)
	certBackupExists, keyBackupExists := certBackupErr == nil, keyBackupErr == nil
	if certBackupErr != nil && !os.IsNotExist(certBackupErr) {
		return fmt.Errorf("inspect certificate rotation backup: %w", certBackupErr)
	}
	if keyBackupErr != nil && !os.IsNotExist(keyBackupErr) {
		return fmt.Errorf("inspect key rotation backup: %w", keyBackupErr)
	}
	if certBackupExists != keyBackupExists {
		return fmt.Errorf("incomplete mTLS rotation backup for %s; manual recovery required", nodeID)
	}
	if !certBackupExists {
		return nil
	}

	for _, path := range []string{certBackup, keyBackup} {
		st, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect rotation backup %s: %w", path, err)
		}
		if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse non-regular rotation backup %s", path)
		}
	}
	if _, hasPending := pending[nodeID]; hasPending {
		currentCert, err := parseCertFile(certPath)
		if err != nil {
			return fmt.Errorf("pending rotation for %s has no valid current certificate: %w", nodeID, err)
		}
		currentKey, err := os.Lstat(keyPath)
		if err != nil || !currentKey.Mode().IsRegular() || currentKey.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("pending rotation for %s has no valid current key", nodeID)
		}
		if serialHex(currentCert.SerialNumber) != pending[nodeID].NewSerial {
			return fmt.Errorf("pending rotation certificate serial mismatch for %s", nodeID)
		}
		if err := os.Remove(certBackup); err != nil {
			return fmt.Errorf("remove completed certificate rotation backup: %w", err)
		}
		if err := os.Remove(keyBackup); err != nil {
			return fmt.Errorf("remove completed key rotation backup: %w", err)
		}
		return nil
	}

	// No durable pending record means promotion did not commit. Restore the
	// previous pair, regardless of whether the process stopped mid-promotion.
	_ = os.Remove(certPath)
	_ = os.Remove(keyPath)
	if err := os.Rename(certBackup, certPath); err != nil {
		return fmt.Errorf("recover previous client certificate: %w", err)
	}
	if err := os.Rename(keyBackup, keyPath); err != nil {
		return fmt.Errorf("recover previous client key: %w", err)
	}
	return nil
}

func RotateClientForPush(nodeID, reason string) (info ClientInfo, ca, cert, key []byte, err error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("empty node ID")
	}
	if reason == "" {
		reason = "rotate"
	}

	pendingMu.Lock()
	defer pendingMu.Unlock()

	pending, err := loadPending()
	if err != nil {
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("load pending rotate: %w", err)
	}
	if _, exists := pending[nodeID]; exists {
		ca, cert, key, err = ReadMaterial(nodeID)
		if err != nil {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("read pending rotation material: %w", err)
		}
		info, err = ReadClientInfo(nodeID)
		if err != nil {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("read pending rotation info: %w", err)
		}
		return info, ca, cert, key, nil
	}

	dir := ClientDir(nodeID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ClientInfo{}, nil, nil, nil, err
	}
	if err := recoverRotationBackup(nodeID, pending); err != nil {
		return ClientInfo{}, nil, nil, nil, err
	}
	certPath := filepath.Join(dir, ClientCertFile)
	keyPath := filepath.Join(dir, ClientKeyFile)
	certBackup := certPath + ".rotate-backup"
	keyBackup := keyPath + ".rotate-backup"

	_, certErr := os.Lstat(certPath)
	_, keyErr := os.Lstat(keyPath)
	certExists, keyExists := certErr == nil, keyErr == nil
	if certErr != nil && !os.IsNotExist(certErr) {
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("inspect old client certificate: %w", certErr)
	}
	if keyErr != nil && !os.IsNotExist(keyErr) {
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("inspect old client key: %w", keyErr)
	}
	if certExists != keyExists {
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("incomplete existing client material for %s; refusing rotation", nodeID)
	}

	oldSerial := ""
	if certExists {
		oldCert, parseErr := parseCertFile(certPath)
		if parseErr != nil {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("parse existing client certificate: %w", parseErr)
		}
		oldSerial = serialHex(oldCert.SerialNumber)
		if oldSerial == "" {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("existing client certificate has empty serial")
		}
		if err := os.Rename(certPath, certBackup); err != nil {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("stage old client certificate: %w", err)
		}
		if err := os.Rename(keyPath, keyBackup); err != nil {
			restoreErr := os.Rename(certBackup, certPath)
			if restoreErr != nil {
				return ClientInfo{}, nil, nil, nil, fmt.Errorf("stage old client key: %w; restore old certificate: %v", err, restoreErr)
			}
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("stage old client key: %w", err)
		}
	}

	restoreOld := func() error {
		var errs []string
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
		if certExists {
			if e := os.Rename(certBackup, certPath); e != nil {
				errs = append(errs, "restore certificate: "+e.Error())
			}
			if e := os.Rename(keyBackup, keyPath); e != nil {
				errs = append(errs, "restore key: "+e.Error())
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("%s", strings.Join(errs, "; "))
		}
		return nil
	}

	ca, cert, key, err = EnsureClientFor(nodeID)
	if err != nil {
		if rbErr := restoreOld(); rbErr != nil {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("issue new client material: %w; rollback incomplete: %v", err, rbErr)
		}
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("issue new client material: %w", err)
	}
	info, err = ReadClientInfo(nodeID)
	if err != nil {
		if rbErr := restoreOld(); rbErr != nil {
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("read new client info: %w; rollback incomplete: %v", err, rbErr)
		}
		return ClientInfo{}, nil, nil, nil, fmt.Errorf("read new client info: %w", err)
	}
	if oldSerial != "" && oldSerial != info.Serial {
		pending[nodeID] = pendingRotate{
			NodeID: nodeID, OldSerial: oldSerial, NewSerial: info.Serial,
			CreatedAt: time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Duration(GraceHours()) * time.Hour).Unix(),
		}
		if err := savePending(pending); err != nil {
			// SaveJSON may report a directory-sync error after rename committed the
			// journal. Never restore the old certificate if that journal is visible.
			visible, loadErr := loadPending()
			if loadErr == nil {
				if committed, ok := visible[nodeID]; ok &&
					committed.OldSerial == oldSerial && committed.NewSerial == info.Serial {
					_ = os.Remove(certBackup)
					_ = os.Remove(keyBackup)
					return ClientInfo{}, nil, nil, nil, fmt.Errorf("pending rotation journal committed but durability confirmation failed: %w", err)
				}
			}
			if rbErr := restoreOld(); rbErr != nil {
				return ClientInfo{}, nil, nil, nil, fmt.Errorf("save pending rotate: %w; rollback incomplete: %v", err, rbErr)
			}
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("save pending rotate: %w", err)
		}
	}
	if certExists {
		_ = os.Remove(certBackup)
		_ = os.Remove(keyBackup)
	}
	return info, ca, cert, key, nil
}

// ConfirmRotate marks new cert in use and revokes old serial.
func ConfirmRotate(nodeID, presentedSerial string) error {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	m, err := loadPending()
	if err != nil {
		return err
	}
	p, ok := m[nodeID]
	if !ok {
		return nil // nothing pending
	}
	presentedSerial = strings.ToLower(strings.TrimSpace(presentedSerial))
	if presentedSerial != "" && presentedSerial != p.NewSerial {
		// still old cert
		return nil
	}
	if p.OldSerial != "" {
		if err := RevokeSerial(p.OldSerial, nodeID, "rotate-confirmed"); err != nil {
			return fmt.Errorf("revoke previous client certificate: %w", err)
		}
	}
	delete(m, nodeID)
	return savePending(m)
}

// PendingRotates returns open grace windows.
func PendingRotates() ([]pendingRotate, error) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	m, err := loadPending()
	if err != nil {
		return nil, err
	}
	out := make([]pendingRotate, 0, len(m))
	for _, p := range m {
		out = append(out, p)
	}
	return out, nil
}

// ReadMaterial returns current client PEM for node (for push endpoint).
func ReadMaterial(nodeID string) (ca, cert, key []byte, err error) {
	ca, err = os.ReadFile(path(CACertFile))
	if err != nil {
		return
	}
	// dual-CA: also append ca-new if present so agent trusts both
	if b, e := os.ReadFile(path("ca-new.crt")); e == nil && len(b) > 0 {
		ca = append(ca, '\n')
		ca = append(ca, b...)
	}
	cert, err = os.ReadFile(filepath.Join(ClientDir(nodeID), ClientCertFile))
	if err != nil {
		return
	}
	key, err = os.ReadFile(filepath.Join(ClientDir(nodeID), ClientKeyFile))
	return
}
