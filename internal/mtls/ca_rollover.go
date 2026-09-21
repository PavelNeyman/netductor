package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"time"
)

const (
	CANewCertFile = "ca-new.crt"
	CANewKeyFile  = "ca-new.key"
	rolloverState = "ca_rollover.json"
)

type rolloverInfo struct {
	StartedAt int64  `json:"started_at"`
	Phase     string `json:"phase"` // "" | dual | done
	NewSerial string `json:"new_serial,omitempty"`
}

func rolloverPath() string { return filepath.Join(Dir(), rolloverState) }

func loadRollover() rolloverInfo {
	var r rolloverInfo
	b, err := os.ReadFile(rolloverPath())
	if err != nil {
		return r
	}
	_ = json.Unmarshal(b, &r)
	return r
}

func saveRollover(r rolloverInfo) error {
	raw, _ := json.MarshalIndent(r, "", "  ")
	return os.WriteFile(rolloverPath(), append(raw, '\n'), 0o600)
}

// CARolloverStart generates ca-new. Both CAs trusted until CARolloverFinish.
func CARolloverStart() error {
	if err := ensureCA(); err != nil {
		return err
	}
	if fileOK(path(CANewCertFile)) {
		return fmt.Errorf("ca-new already exists; finish or abort first")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "netductor-agent-ca-new", Organization: []string{"netductor"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := writeCert(path(CANewCertFile), der); err != nil {
		return err
	}
	if err := writeKey(path(CANewKeyFile), key); err != nil {
		return err
	}
	c, _ := parseCertFile(path(CANewCertFile))
	ser := ""
	if c != nil {
		ser = serialHex(c.SerialNumber)
	}
	return saveRollover(rolloverInfo{StartedAt: time.Now().Unix(), Phase: "dual", NewSerial: ser})
}

// IssueClientFromNewCA issues client cert signed by ca-new.
func IssueClientFromNewCA(nodeID string) (caBundle, cert, key []byte, err error) {
	if !fileOK(path(CANewCertFile)) || !fileOK(path(CANewKeyFile)) {
		return nil, nil, nil, fmt.Errorf("ca-new missing; run mtls rollover start")
	}
	caCert, caKey, err := loadNamedCA(path(CANewCertFile), path(CANewKeyFile))
	if err != nil {
		return nil, nil, nil, err
	}
	dir := ClientDir(nodeID)
	_ = os.MkdirAll(dir, 0o700)
	certPath := filepath.Join(dir, ClientCertFile)
	keyPath := filepath.Join(dir, ClientKeyFile)
	if fileOK(certPath) {
		if c, e := parseCertFile(certPath); e == nil {
			_ = RevokeSerial(serialHex(c.SerialNumber), nodeID, "rollover-reissue")
		}
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
	}
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: "netductor-agent-" + nodeID, Organization: []string{"netductor"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &k.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := writeCert(certPath, der); err != nil {
		return nil, nil, nil, err
	}
	if err := writeKey(keyPath, k); err != nil {
		return nil, nil, nil, err
	}
	return ReadMaterial(nodeID)
}

func loadNamedCA(certP, keyP string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	cb, err := os.ReadFile(certP)
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(cb)
	if block == nil {
		return nil, nil, fmt.Errorf("bad ca cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	kb, err := os.ReadFile(keyP)
	if err != nil {
		return nil, nil, err
	}
	kblock, _ := pem.Decode(kb)
	if kblock == nil {
		return nil, nil, fmt.Errorf("bad ca key")
	}
	key, err := x509.ParseECPrivateKey(kblock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// CARolloverFinish promotes ca-new to primary CA and removes dual trust.
// Call only when all clients use certs from ca-new.
func CARolloverFinish() error {
	if !fileOK(path(CANewCertFile)) {
		return fmt.Errorf("no ca-new")
	}
	// backup old
	_ = os.Rename(path(CACertFile), path("ca-old.crt"))
	_ = os.Rename(path(CAKeyFile), path("ca-old.key"))
	if err := os.Rename(path(CANewCertFile), path(CACertFile)); err != nil {
		return err
	}
	if err := os.Rename(path(CANewKeyFile), path(CAKeyFile)); err != nil {
		return err
	}
	// re-issue server cert under new CA
	_ = os.Remove(path(ServerCertFile))
	_ = os.Remove(path(ServerKeyFile))
	pub := ""
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(Dir()), "public_ip")); err == nil {
		pub = string(b)
	}
	if err := ensureServer(strings.TrimSpace(pub)); err != nil {
		return err
	}
	return saveRollover(rolloverInfo{Phase: "done", StartedAt: time.Now().Unix()})
}

// CARolloverAbort removes ca-new without promoting.
func CARolloverAbort() error {
	_ = os.Remove(path(CANewCertFile))
	_ = os.Remove(path(CANewKeyFile))
	return saveRollover(rolloverInfo{})
}

// CARolloverStatus returns phase and whether dual CA files exist.
func CARolloverStatus() map[string]any {
	r := loadRollover()
	return map[string]any{
		"phase":        r.Phase,
		"started_at":   r.StartedAt,
		"new_serial":   r.NewSerial,
		"ca_new_ready": fileOK(path(CANewCertFile)),
		"dual_trust":   fileOK(path(CANewCertFile)) && fileOK(path(CACertFile)),
	}
}

// AppendExtraCAs adds ca-new to a cert pool if present.
func AppendExtraCAs(pool *x509.CertPool) {
	if b, err := os.ReadFile(path(CANewCertFile)); err == nil {
		pool.AppendCertsFromPEM(b)
	}
}
