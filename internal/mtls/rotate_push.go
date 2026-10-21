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
		if p.ExpiresAt > 0 && now > p.ExpiresAt {
			_ = RevokeSerial(p.OldSerial, p.NodeID, "grace-expired")
			continue
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

// RotateClientForPush re-issues cert, keeps old serial valid until Confirm or grace.
// Returns material for push + pending metadata.
func RotateClientForPush(nodeID, reason string) (info ClientInfo, ca, cert, key []byte, err error) {
	if reason == "" {
		reason = "rotate"
	}
	certPath := filepath.Join(ClientDir(nodeID), ClientCertFile)
	keyPath := filepath.Join(ClientDir(nodeID), ClientKeyFile)
	var oldSerial string
	if fileOK(certPath) {
		if c, e := parseCertFile(certPath); e == nil {
			oldSerial = serialHex(c.SerialNumber)
		}
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
	}
	ca, cert, key, err = EnsureClientFor(nodeID)
	if err != nil {
		return ClientInfo{}, nil, nil, nil, err
	}
	info, err = ReadClientInfo(nodeID)
	if err != nil {
		return ClientInfo{}, nil, nil, nil, err
	}
	if oldSerial != "" && oldSerial != info.Serial {
		pendingMu.Lock()
		m, errLoad := loadPending()
		if errLoad != nil {
			pendingMu.Unlock()
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("load pending rotate: %w", errLoad)
		}
		m[nodeID] = pendingRotate{
			NodeID: nodeID, OldSerial: oldSerial, NewSerial: info.Serial,
			CreatedAt: time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Duration(GraceHours()) * time.Hour).Unix(),
		}
		if err := savePending(m); err != nil {
			pendingMu.Unlock()
			return ClientInfo{}, nil, nil, nil, fmt.Errorf("save pending rotate: %w", err)
		}
		pendingMu.Unlock()
		// do NOT revoke old yet — agent still uses it to fetch material
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
		_ = RevokeSerial(p.OldSerial, nodeID, "rotate-confirmed")
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
