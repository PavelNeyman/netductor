package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
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

type rolloverClientState struct {
	OldSerial   string `json:"old_serial,omitempty"`
	NewSerial   string `json:"new_serial"`
	ConfirmedAt int64  `json:"confirmed_at,omitempty"`
}

type rolloverInfo struct {
	StartedAt int64                           `json:"started_at"`
	Phase     string                          `json:"phase"` // dual | done | aborted
	NewSerial string                          `json:"new_serial,omitempty"`
	Issued    map[string]rolloverClientState  `json:"issued,omitempty"`
}

func rolloverPath() string { return filepath.Join(Dir(), rolloverState) }

func loadRollover() (rolloverInfo, error) {
	var r rolloverInfo
	if err := statefile.LoadJSON(rolloverPath(), &r); err != nil {
		if statefile.IsNotExist(err) {
			return rolloverInfo{}, nil
		}
		return rolloverInfo{}, fmt.Errorf("load CA rollover journal: %w", err)
	}
	switch r.Phase {
	case "dual", "done", "aborted":
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
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase == "dual" {
		return fmt.Errorf("CA rollover is already in progress; finish or abort the existing operation")
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
		_ = os.Remove(path(CANewCertFile))
		_ = os.Remove(path(CANewKeyFile))
		return fmt.Errorf("write new CA certificate: %w", err)
	}
	if err := writeKey(path(CANewKeyFile), key); err != nil {
		_ = os.Remove(path(CANewCertFile))
		_ = os.Remove(path(CANewKeyFile))
		return fmt.Errorf("write new CA key: %w", err)
	}
	c, err := parseCertFile(path(CANewCertFile))
	if err != nil {
		_ = os.Remove(path(CANewCertFile))
		_ = os.Remove(path(CANewKeyFile))
		return fmt.Errorf("verify new CA certificate: %w", err)
	}
	ser := serialHex(c.SerialNumber)
	if ser == "" {
		_ = os.Remove(path(CANewCertFile))
		_ = os.Remove(path(CANewKeyFile))
		return fmt.Errorf("new CA certificate has no serial")
	}
	return saveRollover(rolloverInfo{StartedAt: time.Now().Unix(), Phase: "dual", NewSerial: ser})
}

// IssueClientFromNewCA issues client cert signed by ca-new.
func IssueClientFromNewCA(nodeID string) (caBundle, cert, key []byte, err error) {
	if strings.TrimSpace(nodeID) == "" || strings.ContainsAny(nodeID, "/\\") || nodeID == "." || nodeID == ".." {
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
	if issued, ok := rollover.Issued[nodeID]; ok {
		current, err := ReadClientInfo(nodeID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load previously issued rollover identity: %w", err)
		}
		if !strings.EqualFold(current.Serial, issued.NewSerial) {
			return nil, nil, nil, fmt.Errorf("current client certificate for %q does not match its recorded rollover identity", nodeID)
		}
		return ReadMaterial(nodeID)
	}
	dir := ClientDir(nodeID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, nil, fmt.Errorf("create client directory: %w", err)
	}
	certPath := filepath.Join(dir, ClientCertFile)
	keyPath := filepath.Join(dir, ClientKeyFile)
	oldSerial := ""
	if fileOK(certPath) {
		oldInfo, err := ReadClientInfo(nodeID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read previous client identity: %w", err)
		}
		oldSerial = oldInfo.Serial
	}
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
	newCertBlock, _ := pem.Decode(materialCert)
	if newCertBlock == nil {
		rollback()
		return nil, nil, nil, fmt.Errorf("promoted client certificate is not PEM")
	}
	newCert, err := x509.ParseCertificate(newCertBlock.Bytes)
	if err != nil {
		rollback()
		return nil, nil, nil, fmt.Errorf("parse promoted client certificate: %w", err)
	}
	rollover.Issued = ensureRolloverIssued(rollover.Issued)
	newSerial := serialHex(newCert.SerialNumber)
	rollover.Issued[nodeID] = rolloverClientState{OldSerial: oldSerial, NewSerial: newSerial}
	if err := saveRollover(rollover); err != nil {
		// SaveJSON may report a directory-sync error after the atomic rename.
		// If the journal entry is already visible, preserve the matching new
		// pair so a retry can safely deliver and confirm it.
		persisted, loadErr := loadRollover()
		if loadErr == nil {
			if state, ok := persisted.Issued[nodeID]; ok && state.NewSerial == newSerial {
				return materialCA, materialCert, materialKey, nil
			}
		}
		rollback()
		return nil, nil, nil, fmt.Errorf("persist issued CA rollover identity: %w", err)
	}
	// The old serial remains valid until the Secondary presents this exact
	// replacement certificate on an authenticated TLS heartbeat.
	return materialCA, materialCert, materialKey, nil
}

func parseSerial(serial string) *big.Int {
	n, ok := new(big.Int).SetString(strings.TrimSpace(serial), 16)
	if !ok {
		return nil
	}
	return n
}

func ensureRolloverIssued(in map[string]rolloverClientState) map[string]rolloverClientState {
	if in == nil {
		return make(map[string]rolloverClientState)
	}
	return in
}

// ConfirmCARolloverClient records adoption only when the server-observed TLS
// serial matches the serial issued for this node. Old serial revocation follows
// the durable confirmation and is safe to retry.
func ConfirmCARolloverClient(nodeID, peerSerial string) (bool, error) {
	peerSerial = strings.ToLower(strings.TrimSpace(peerSerial))
	if nodeID == "" || peerSerial == "" {
		return false, nil
	}
	rollover, err := loadRollover()
	if err != nil {
		return false, err
	}
	if rollover.Phase != "dual" {
		return false, nil
	}
	state, ok := rollover.Issued[nodeID]
	if !ok || !strings.EqualFold(state.NewSerial, peerSerial) {
		return false, nil
	}
	if state.ConfirmedAt == 0 {
		state.ConfirmedAt = time.Now().Unix()
		rollover.Issued[nodeID] = state
		if err := saveRollover(rollover); err != nil {
			return false, fmt.Errorf("persist Secondary CA adoption: %w", err)
		}
	}
	if state.OldSerial != "" {
		if err := RevokeSerial(state.OldSerial, nodeID, "ca-rollover-adopted"); err != nil {
			return false, fmt.Errorf("revoke previous Secondary certificate after adoption: %w", err)
		}
	}
	return true, nil
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
	for nodeID, state := range rollover.Issued {
		if state.ConfirmedAt == 0 {
			return fmt.Errorf("CA rollover is not complete: Secondary %q has not confirmed its new TLS certificate", nodeID)
		}
		if state.OldSerial != "" {
			oldSerial := parseSerial(state.OldSerial)
			if oldSerial == nil {
				return fmt.Errorf("invalid old certificate serial for %q", nodeID)
			}
			revoked, err := IsRevoked(oldSerial)
			if err != nil {
				return fmt.Errorf("check old certificate revocation for %q: %w", nodeID, err)
			}
			if !revoked {
				return fmt.Errorf("CA rollover is not complete: old certificate for %q is not revoked", nodeID)
			}
		}
	}
	if !fileOK(path(CANewCertFile)) || !fileOK(path(CANewKeyFile)) {
		return fmt.Errorf("new CA certificate/key pair is incomplete")
	}
	oldCA, err := parseCertFile(path(CACertFile))
	if err != nil {
		return fmt.Errorf("read current CA before promotion: %w", err)
	}
	clientsDir := filepath.Join(Dir(), "clients")
	clients, err := os.ReadDir(clientsDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("scan Secondary certificates before promotion: %w", err)
	}
	for _, entry := range clients {
		if !entry.IsDir() {
			continue
		}
		clientCert, err := parseCertFile(filepath.Join(clientsDir, entry.Name(), ClientCertFile))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("inspect client certificate %q before promotion: %w", entry.Name(), err)
		}
		if clientCert.Issuer.String() == oldCA.Subject.String() {
			return fmt.Errorf("CA rollover is not complete: active client certificate for %q still uses the old CA", entry.Name())
		}
	}
	// Promotion still replaces the CA certificate, CA key, and server
	// certificate/key as separate files. Keep it disabled until those changes
	// are covered by a durable crash-recovery transaction.
	return fmt.Errorf("CA rollover clients are confirmed, but crash-safe CA/server promotion is not implemented")
}

// CARolloverAbort removes ca-new only when no client certificate has been
// issued by it. Once a client has received a new-CA identity, deleting that CA
// would make the identity unverifiable and can strand the Secondary.
func CARolloverAbort() error {
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase == "done" {
		return fmt.Errorf("cannot abort a completed CA rollover")
	}
	if len(rollover.Issued) > 0 {
		return fmt.Errorf("cannot abort CA rollover after new-CA client identities have been issued")
	}
	if fileOK(path(CANewCertFile)) {
		newCA, err := parseCertFile(path(CANewCertFile))
		if err != nil {
			return fmt.Errorf("read new CA before abort: %w", err)
		}
		root := filepath.Join(Dir(), "clients")
		entries, err := os.ReadDir(root)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("scan client certificates before abort: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			clientCert, err := parseCertFile(filepath.Join(root, entry.Name(), ClientCertFile))
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("inspect client certificate %q before abort: %w", entry.Name(), err)
			}
			if clientCert.Issuer.String() == newCA.Subject.String() {
				return fmt.Errorf("cannot abort CA rollover: client certificate for %q has already been issued by the new CA", entry.Name())
			}
		}
	}
	if err := os.Remove(path(CANewCertFile)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove new CA certificate: %w", err)
	}
	if err := os.Remove(path(CANewKeyFile)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove new CA key: %w", err)
	}
	return saveRollover(rolloverInfo{Phase: "aborted", StartedAt: time.Now().Unix()})
}

// CARolloverStatus returns phase and whether dual CA files exist.
func CARolloverStatus() map[string]any {
	r, err := loadRollover()
	if err != nil {
		return map[string]any{"error": err.Error(), "ca_new_ready": fileOK(path(CANewCertFile))}
	}
	confirmed := 0
	for _, client := range r.Issued {
		if client.ConfirmedAt > 0 {
			confirmed++
		}
	}
	return map[string]any{
		"phase":             r.Phase,
		"started_at":        r.StartedAt,
		"new_serial":        r.NewSerial,
		"issued_clients":    len(r.Issued),
		"confirmed_clients": confirmed,
		"ca_new_ready":      fileOK(path(CANewCertFile)),
		"dual_trust":        fileOK(path(CANewCertFile)) && fileOK(path(CACertFile)),
	}
}

// AppendExtraCAs adds ca-new to a cert pool if present.
func AppendExtraCAs(pool *x509.CertPool) {
	if b, err := os.ReadFile(path(CANewCertFile)); err == nil {
		pool.AppendCertsFromPEM(b)
	}
}
