package mtls

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const revokeFile = "revoked.json"

type revokeEntry struct {
	Serial    string `json:"serial"` // hex
	NodeID    string `json:"node_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	RevokedAt int64  `json:"revoked_at"`
}

type revokeFileData struct {
	Entries []revokeEntry `json:"entries"`
}

var revokeMu sync.Mutex

func revokePath() string {
	return filepath.Join(Dir(), revokeFile)
}

func loadRevoked() (map[string]revokeEntry, error) {
	out := map[string]revokeEntry{}
	b, err := os.ReadFile(revokePath())
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	var d revokeFileData
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	for _, e := range d.Entries {
		out[strings.ToLower(e.Serial)] = e
	}
	return out, nil
}

func saveRevoked(m map[string]revokeEntry) error {
	_ = os.MkdirAll(Dir(), 0o700)
	d := revokeFileData{}
	for _, e := range m {
		d.Entries = append(d.Entries, e)
	}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := revokePath() + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, revokePath())
}

func serialHex(s *big.Int) string {
	if s == nil {
		return ""
	}
	return strings.ToLower(s.Text(16))
}

// IsRevoked reports whether client cert serial is on the revoke list.
func IsRevoked(serial *big.Int) bool {
	if serial == nil {
		return false
	}
	revokeMu.Lock()
	defer revokeMu.Unlock()
	m, err := loadRevoked()
	if err != nil {
		return false
	}
	_, ok := m[serialHex(serial)]
	return ok
}

// RevokeSerial adds serial to the revoke list.
func RevokeSerial(serialHexStr, nodeID, reason string) error {
	serialHexStr = strings.ToLower(strings.TrimSpace(serialHexStr))
	if serialHexStr == "" {
		return fmt.Errorf("empty serial")
	}
	revokeMu.Lock()
	defer revokeMu.Unlock()
	m, err := loadRevoked()
	if err != nil {
		return err
	}
	m[serialHexStr] = revokeEntry{
		Serial: serialHexStr, NodeID: nodeID, Reason: reason,
		RevokedAt: time.Now().Unix(),
	}
	return saveRevoked(m)
}

// RevokeNodeCert reads current client cert for node and revokes its serial.
func RevokeNodeCert(nodeID, reason string) (serial string, err error) {
	info, err := ReadClientInfo(nodeID)
	if err != nil {
		return "", err
	}
	if err := RevokeSerial(info.Serial, nodeID, reason); err != nil {
		return "", err
	}
	return info.Serial, nil
}

// ListRevoked returns revoke entries.
func ListRevoked() ([]revokeEntry, error) {
	revokeMu.Lock()
	defer revokeMu.Unlock()
	m, err := loadRevoked()
	if err != nil {
		return nil, err
	}
	out := make([]revokeEntry, 0, len(m))
	for _, e := range m {
		out = append(out, e)
	}
	return out, nil
}

// ClientInfo is observability for a per-node client cert.
type ClientInfo struct {
	NodeID    string    `json:"node_id"`
	Serial    string    `json:"serial"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	Subject   string    `json:"subject"`
	Path      string    `json:"path"`
	DaysLeft  int       `json:"days_left"`
	Revoked   bool      `json:"revoked"`
}

func parseCertFile(certPath string) (*x509.Certificate, error) {
	b, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("no PEM in %s", certPath)
	}
	return x509.ParseCertificate(block.Bytes)
}

// ReadClientInfo loads client cert metadata for nodeID.
func ReadClientInfo(nodeID string) (ClientInfo, error) {
	certPath := filepath.Join(ClientDir(nodeID), ClientCertFile)
	c, err := parseCertFile(certPath)
	if err != nil {
		return ClientInfo{}, err
	}
	days := int(time.Until(c.NotAfter).Hours() / 24)
	info := ClientInfo{
		NodeID: nodeID, Serial: serialHex(c.SerialNumber),
		NotBefore: c.NotBefore, NotAfter: c.NotAfter,
		Subject: c.Subject.CommonName, Path: certPath, DaysLeft: days,
		Revoked: IsRevoked(c.SerialNumber),
	}
	return info, nil
}

// ListClientCerts scans secrets/mtls/clients/*.
func ListClientCerts() ([]ClientInfo, error) {
	root := filepath.Join(Dir(), "clients")
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ClientInfo
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		info, err := ReadClientInfo(e.Name())
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

// PlaneCertInfo for CA/server/default client expiry checks.
type PlaneCertInfo struct {
	Name     string    `json:"name"`
	NotAfter time.Time `json:"not_after"`
	DaysLeft int       `json:"days_left"`
	Serial   string    `json:"serial"`
}

func readPlaneCert(name, certPath string) (PlaneCertInfo, error) {
	c, err := parseCertFile(certPath)
	if err != nil {
		return PlaneCertInfo{}, err
	}
	return PlaneCertInfo{
		Name: name, NotAfter: c.NotAfter, DaysLeft: int(time.Until(c.NotAfter).Hours() / 24),
		Serial: serialHex(c.SerialNumber),
	}, nil
}

// ListPlaneCerts returns CA, server, default client expiry.
func ListPlaneCerts() []PlaneCertInfo {
	var out []PlaneCertInfo
	for _, pair := range []struct{ n, p string }{
		{"ca", path(CACertFile)},
		{"server", path(ServerCertFile)},
		{"client-default", path(ClientCertFile)},
	} {
		if info, err := readPlaneCert(pair.n, pair.p); err == nil {
			out = append(out, info)
		}
	}
	return out
}

// RotateClientFor re-issues client cert for nodeID and revokes the previous serial.
func RotateClientFor(nodeID, reason string) (ClientInfo, error) {
	if reason == "" {
		reason = "rotate"
	}
	certPath := filepath.Join(ClientDir(nodeID), ClientCertFile)
	keyPath := filepath.Join(ClientDir(nodeID), ClientKeyFile)
	var oldSerial string
	if fileOK(certPath) {
		if c, err := parseCertFile(certPath); err == nil {
			oldSerial = serialHex(c.SerialNumber)
		}
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
	}
	if _, _, _, err := EnsureClientFor(nodeID); err != nil {
		return ClientInfo{}, err
	}
	if oldSerial != "" {
		_ = RevokeSerial(oldSerial, nodeID, reason)
	}
	return ReadClientInfo(nodeID)
}
