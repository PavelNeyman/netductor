package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/statefile"
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

func loadRollover() (rolloverInfo, error) {
	var r rolloverInfo
	if err := statefile.LoadJSON(rolloverPath(), &r); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return rolloverInfo{}, nil
		}
		return rolloverInfo{}, fmt.Errorf("load CA rollover journal: %w", err)
	}
	switch r.Phase {
	case "dual", "done":
		if r.StartedAt <= 0 || (r.Phase == "dual" && r.NewSerial == "") {
			return rolloverInfo{}, fmt.Errorf("invalid CA rollover journal: incomplete %q phase", r.Phase)
		}
	default:
		return rolloverInfo{}, fmt.Errorf("invalid CA rollover journal phase %q", r.Phase)
	}
	return r, nil
}

func saveRollover(r rolloverInfo) error {
	return statefile.SaveJSON(rolloverPath(), r)
}

// CARolloverStart generates ca-new. Both CAs trusted until CARolloverFinish.
func CARolloverStart() error {
	if err := ensureCA(); err != nil {
		return err
	}
	if _, err := loadRollover(); err != nil {
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
	if strings.TrimSpace(nodeID) == "" || strings.ContainsAny(nodeID, "/\\\\") || nodeID == "." || nodeID == ".." {
		return nil, nil, nil, fmt.Errorf("invalid node ID")
	}
	if !fileOK(path(CANewCertFile)) || !fileOK(path(CANewKeyFile)) {
		return nil, nil, nil, fmt.Errorf("ca-new missing; run mtls rollover start")
	}
	rollover, err := loadRollover()
	if err != nil {
		return nil, nil, nil, err
	}
	if rollover.Phase != "dual" || rollover.NewSerial == "" {
		return nil, nil, nil, fmt.Errorf("CA rollover journal is not in a valid dual-trust phase")
	}
	caCert, caKey, err := loadNamedCA(path(CANewCertFile), path(CANewKeyFile))
	if err != nil {
		return nil, nil, nil, err
	}
	if serialHex(caCert.SerialNumber) != rollover.NewSerial {
		return nil, nil, nil, fmt.Errorf("ca-new certificate does not match rollover journal")
	}
	dir := ClientDir(nodeID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, nil, fmt.Errorf("create client directory: %w", err)
	}
	certPath := filepath.Join(dir, ClientCertFile)
	keyPath := filepath.Join(dir, ClientKeyFile)
	tmpCert := certPath + ".rollover-new"
	tmpKey := keyPath + ".rollover-new"
	backupCert := certPath + ".rollover-old"
	backupKey := keyPath + ".rollover-old"
	for _, p := range []string{tmpCert, tmpKey} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return nil, nil, nil, fmt.Errorf("remove stale rollover staging file: %w", err)
		}
	}
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject: pkix.Name{CommonName: "netductor-agent-" + nodeID, Organization: []string{"netductor"}},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter: time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &k.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := writeCert(tmpCert, der); err != nil {
		return nil, nil, nil, err
	}
	if err := writeKey(tmpKey, k); err != nil {
		_ = os.Remove(tmpCert)
		return nil, nil, nil, err
	}
	// Keep the previous identity valid: delivery and adoption by the Secondary
	// are not yet acknowledged by the current protocol. Never revoke it here.
	if fileOK(backupCert) || fileOK(backupKey) {
		// A complete active pair means the prior promotion reached its commit
		// point; stale backups are cleanup residue. Otherwise restore the old
		// pair before retrying so an interrupted promotion cannot strand it.
		if fileOK(certPath) && fileOK(keyPath) {
			if err := os.Remove(backupCert); err != nil && !os.IsNotExist(err) {
				return nil, nil, nil, fmt.Errorf("remove stale rollover certificate backup: %w", err)
			}
			if err := os.Remove(backupKey); err != nil && !os.IsNotExist(err) {
				return nil, nil, nil, fmt.Errorf("remove stale rollover key backup: %w", err)
			}
		} else {
			// Restore only files that were actually moved to backup. A crash
			// between the two backup renames must not delete the untouched half.
			if fileOK(backupCert) {
				_ = os.Remove(certPath)
				if err := os.Rename(backupCert, certPath); err != nil {
					return nil, nil, nil, fmt.Errorf("recover previous client certificate: %w", err)
				}
			}
			if fileOK(backupKey) {
				_ = os.Remove(keyPath)
				if err := os.Rename(backupKey, keyPath); err != nil {
					return nil, nil, nil, fmt.Errorf("recover previous client key: %w", err)
				}
			}
		}
	}
	hadCert, hadKey := fileOK(certPath), fileOK(keyPath)
	if hadCert {
		if err := os.Rename(certPath, backupCert); err != nil {
			_ = os.Remove(tmpCert)
			_ = os.Remove(tmpKey)
			return nil, nil, nil, fmt.Errorf("stage previous client certificate: %w", err)
		}
	}
	if hadKey {
		if err := os.Rename(keyPath, backupKey); err != nil {
			if hadCert { _ = os.Rename(backupCert, certPath) }
			_ = os.Remove(tmpCert)
			_ = os.Remove(tmpKey)
			return nil, nil, nil, fmt.Errorf("stage previous client key: %w", err)
		}
	}
	rollback := func() {
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
		if hadCert { _ = os.Rename(backupCert, certPath) }
		if hadKey { _ = os.Rename(backupKey, keyPath) }
		_ = os.Remove(tmpCert)
		_ = os.Remove(tmpKey)
	}
	if err := os.Rename(tmpCert, certPath); err != nil {
		rollback()
		return nil, nil, nil, fmt.Errorf("promote rollover client certificate: %w", err)
	}
	if err := os.Rename(tmpKey, keyPath); err != nil {
		rollback()
		return nil, nil, nil, fmt.Errorf("promote rollover client key: %w", err)
	}
	materialCA, materialCert, materialKey, err := ReadMaterial(nodeID)
	if err != nil {
		rollback()
		return nil, nil, nil, fmt.Errorf("read promoted rollover material: %w", err)
	}
	// Retain the old pair until an explicit, separately verified adoption
	// protocol can revoke it. Cleanup is intentionally deferred.
	return materialCA, materialCert, materialKey, nil
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
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase != "dual" {
		return fmt.Errorf("CA rollover is not in dual-trust phase")
	}
	// The current Secondary protocol has no durable acknowledgement that the
	// agent reconnected using its new-CA certificate. Promoting the CA here
	// would remove trust for potentially offline clients and lock them out.
	return fmt.Errorf("CA rollover finish is blocked: per-Secondary new-certificate TLS acknowledgements are not implemented")
	if !fileOK(path(CANewCertFile)) {
		return fmt.Errorf("no ca-new")
	}
	if !fileOK(path(CANewKeyFile)) {
		return fmt.Errorf("no ca-new key")
	}
	// backup old (must succeed before promote)
	if err := os.Rename(path(CACertFile), path("ca-old.crt")); err != nil {
		return fmt.Errorf("backup ca.crt: %w", err)
	}
	if err := os.Rename(path(CAKeyFile), path("ca-old.key")); err != nil {
		// try restore cert
		_ = os.Rename(path("ca-old.crt"), path(CACertFile))
		return fmt.Errorf("backup ca.key: %w", err)
	}
	if err := os.Rename(path(CANewCertFile), path(CACertFile)); err != nil {
		_ = os.Rename(path("ca-old.crt"), path(CACertFile))
		_ = os.Rename(path("ca-old.key"), path(CAKeyFile))
		return fmt.Errorf("promote ca-new.crt: %w", err)
	}
	if err := os.Rename(path(CANewKeyFile), path(CAKeyFile)); err != nil {
		_ = os.Rename(path(CACertFile), path(CANewCertFile))
		_ = os.Rename(path("ca-old.crt"), path(CACertFile))
		_ = os.Rename(path("ca-old.key"), path(CAKeyFile))
		return fmt.Errorf("promote ca-new.key: %w", err)
	}
	// Backup server cert before re-issue so ensureServer failure can restore.
	if err := os.Rename(path(ServerCertFile), path("server-old.crt")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("backup server.crt: %w", err)
	}
	if err := os.Rename(path(ServerKeyFile), path("server-old.key")); err != nil && !os.IsNotExist(err) {
		_ = os.Rename(path("server-old.crt"), path(ServerCertFile))
		return fmt.Errorf("backup server.key: %w", err)
	}
	pub := ""
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(Dir()), "public_ip")); err == nil {
		pub = string(b)
	}
	if err := ensureServer(strings.TrimSpace(pub)); err != nil {
		var rb []string
		for _, pair := range [][2]string{
			{path(CACertFile), path(CANewCertFile)},
			{path(CAKeyFile), path(CANewKeyFile)},
			{path("ca-old.crt"), path(CACertFile)},
			{path("ca-old.key"), path(CAKeyFile)},
			{path("server-old.crt"), path(ServerCertFile)},
			{path("server-old.key"), path(ServerKeyFile)},
		} {
			if e := os.Rename(pair[0], pair[1]); e != nil && !os.IsNotExist(e) {
				rb = append(rb, e.Error())
			}
		}
		_ = saveRollover(rolloverInfo{Phase: "dual", StartedAt: time.Now().Unix()})
		msg := fmt.Sprintf("ensureServer after CA promote: %v", err)
		if len(rb) > 0 {
			msg += "; rollback errors: " + strings.Join(rb, "; ")
		} else {
			msg += " (rolled back to previous CA/server)"
		}
		return fmt.Errorf("%s", msg)
	}
	_ = os.Remove(path("server-old.crt"))
	_ = os.Remove(path("server-old.key"))
	if err := saveRollover(rolloverInfo{Phase: "done", StartedAt: time.Now().Unix()}); err != nil {
		return fmt.Errorf("CA promoted and server re-issued but rollover state save failed: %w", err)
	}
	return nil
}

// CARolloverAbort removes ca-new without promoting.
func CARolloverAbort() error {
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase == "done" {
		return fmt.Errorf("cannot abort a completed CA rollover")
	}
	_ = os.Remove(path(CANewCertFile))
	_ = os.Remove(path(CANewKeyFile))
	return saveRollover(rolloverInfo{})
}

// CARolloverStatus returns phase and whether dual CA files exist.
func CARolloverStatus() map[string]any {
	r, err := loadRollover()
	if err != nil {
		return map[string]any{"error": err.Error(), "ca_new_ready": fileOK(path(CANewCertFile))}
	}
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
