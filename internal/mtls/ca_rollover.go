package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
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

var rolloverMu sync.Mutex

func loadRollover() (rolloverInfo, error) {
	var r rolloverInfo
	if err := statefile.LoadJSON(rolloverPath(), &r); err != nil {
		if statefile.IsNotExist(err) {
			return rolloverInfo{}, nil
		}
		return rolloverInfo{}, fmt.Errorf("load CA rollover journal: %w", err)
	}
	switch r.Phase {
	case "dual", "done", "aborted", "promoting":
		if r.StartedAt <= 0 || ((r.Phase == "dual" || r.Phase == "promoting") && r.NewSerial == "") {
			return rolloverInfo{}, fmt.Errorf("invalid CA rollover journal: incomplete %q phase", r.Phase)
		}
	default:
		return rolloverInfo{}, fmt.Errorf("invalid CA rollover journal phase %q", r.Phase)
	}
	for nodeID, client := range r.Issued {
		if strings.TrimSpace(nodeID) == "" || parseSerial(client.NewSerial) == nil {
			return rolloverInfo{}, fmt.Errorf("invalid CA rollover journal client entry %q", nodeID)
		}
		if client.OldSerial != "" && parseSerial(client.OldSerial) == nil {
			return rolloverInfo{}, fmt.Errorf("invalid old serial in CA rollover journal for %q", nodeID)
		}
		if client.ConfirmedAt < 0 {
			return rolloverInfo{}, fmt.Errorf("invalid confirmation timestamp in CA rollover journal for %q", nodeID)
		}
	}
	return r, nil
}

func saveRollover(r rolloverInfo) error {
	return statefile.SaveJSON(rolloverPath(), r)
}

// CARolloverStart generates ca-new. Both CAs trusted until CARolloverFinish.
func CARolloverStart() error {
	rolloverMu.Lock()
	defer rolloverMu.Unlock()
	if err := recoverCARolloverPromotion(); err != nil {
		return fmt.Errorf("recover prior CA promotion: %w", err)
	}

	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase == "dual" {
		return fmt.Errorf("CA rollover is already in progress; finish or abort the existing operation")
	}
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
	rolloverMu.Lock()
	defer rolloverMu.Unlock()

	if nodeID == "" || strings.TrimSpace(nodeID) != nodeID {
		return nil, nil, nil, fmt.Errorf("invalid node ID")
	}
	for _, r := range nodeID {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return nil, nil, nil, fmt.Errorf("invalid node ID")
		}
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
		if current.Revoked {
			return nil, nil, nil, fmt.Errorf("recorded rollover certificate for %q is revoked or revocation state is unavailable", nodeID)
		}
		caBundle, clientCert, clientKey, err := ReadMaterial(nodeID)
		if err != nil {
			return nil, nil, nil, err
		}
		if _, err := tls.X509KeyPair(clientCert, clientKey); err != nil {
			return nil, nil, nil, fmt.Errorf("recorded rollover certificate/key pair is invalid: %w", err)
		}
		return caBundle, clientCert, clientKey, nil
	}
	dir := ClientDir(nodeID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, nil, fmt.Errorf("create client directory: %w", err)
	}
	certPath := filepath.Join(dir, ClientCertFile)
	keyPath := filepath.Join(dir, ClientKeyFile)
	if fileOK(certPath) != fileOK(keyPath) {
		return nil, nil, nil, fmt.Errorf("existing client certificate/key pair for %q is incomplete", nodeID)
	}
	oldSerial := ""
	if fileOK(certPath) {
		oldInfo, err := ReadClientInfo(nodeID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read previous client identity: %w", err)
		}
		if oldInfo.Revoked {
			return nil, nil, nil, fmt.Errorf("previous client certificate for %q is revoked or revocation state is unavailable", nodeID)
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
	if !ok || n.Sign() <= 0 {
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
	rolloverMu.Lock()
	defer rolloverMu.Unlock()

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
// CARolloverValidateClients ensures every registered Secondary supplied by
// the operator has a journaled and confirmed new-CA identity before promotion.
func CARolloverValidateClients(nodeIDs []string) error {
	rolloverMu.Lock()
	defer rolloverMu.Unlock()
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase != "dual" {
		return fmt.Errorf("CA rollover is not in dual-trust phase")
	}
	for _, nodeID := range nodeIDs {
		state, ok := rollover.Issued[nodeID]
		if !ok || state.ConfirmedAt == 0 {
			return fmt.Errorf("registered Secondary %q has not confirmed its new-CA identity", nodeID)
		}
		info, err := ReadClientInfo(nodeID)
		if err != nil {
			return fmt.Errorf("read registered Secondary %q certificate: %w", nodeID, err)
		}
		if info.Revoked || !strings.EqualFold(info.Serial, state.NewSerial) {
			return fmt.Errorf("registered Secondary %q does not have its confirmed active certificate", nodeID)
		}
	}
	return nil
}

func CARolloverValidateObservedClients(observed map[string]string) error {
	rolloverMu.Lock()
	defer rolloverMu.Unlock()
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	if rollover.Phase != "dual" {
		return fmt.Errorf("CA rollover is not in dual-trust phase")
	}
	for nodeID, peerSerial := range observed {
		state, ok := rollover.Issued[nodeID]
		if !ok || state.ConfirmedAt == 0 {
			return fmt.Errorf("registered agent %q has not confirmed its new-CA identity", nodeID)
		}
		if peerSerial == "" || !strings.EqualFold(peerSerial, state.NewSerial) {
			return fmt.Errorf("latest observed TLS identity for %q does not match the issued certificate", nodeID)
		}
		info, err := ReadClientInfo(nodeID)
		if err != nil {
			return fmt.Errorf("read registered agent %q certificate: %w", nodeID, err)
		}
		if info.Revoked || !strings.EqualFold(info.Serial, state.NewSerial) {
			return fmt.Errorf("registered agent %q does not have its confirmed active certificate", nodeID)
		}
	}
	return nil
}

func CARolloverFinish() error {
	rolloverMu.Lock()
	defer rolloverMu.Unlock()
	if err := recoverCARolloverPromotion(); err != nil {
		return fmt.Errorf("recover prior CA promotion: %w", err)
	}
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
		activeCert, err := parseCertFile(filepath.Join(ClientDir(nodeID), ClientCertFile))
		if err != nil {
			return fmt.Errorf("load active client certificate for %q: %w", nodeID, err)
		}
		if serialHex(activeCert.SerialNumber) != state.NewSerial {
			return fmt.Errorf("active client certificate for %q does not match the rollover journal", nodeID)
		}
		if state.OldSerial != "" {
			oldSerial := parseSerial(state.OldSerial)
			if oldSerial == nil {
				return fmt.Errorf("invalid old certificate serial for %q", nodeID)
			}
			backupPath := filepath.Join(ClientDir(nodeID), ClientCertFile+".rollover-old")
			backupCert, err := parseCertFile(backupPath)
			if err != nil {
				return fmt.Errorf("load previous client certificate backup for %q: %w", nodeID, err)
			}
			if serialHex(backupCert.SerialNumber) != state.OldSerial {
				return fmt.Errorf("previous client certificate backup for %q does not match the rollover journal", nodeID)
			}
			backupKeyPath := filepath.Join(ClientDir(nodeID), ClientKeyFile+".rollover-old")
			if _, err := os.Lstat(backupKeyPath); err != nil {
				return fmt.Errorf("load previous client key backup for %q: %w", nodeID, err)
			}
			if _, err := tls.LoadX509KeyPair(backupPath, backupKeyPath); err != nil {
				return fmt.Errorf("validate previous client certificate/key backup for %q: %w", nodeID, err)
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
	oldCA, oldCAKey, err := loadNamedCA(path(CACertFile), path(CAKeyFile))
	if err != nil {
		return fmt.Errorf("read current CA pair before promotion: %w", err)
	}
	if err := validateCAKeyPair(oldCA, oldCAKey); err != nil {
		return fmt.Errorf("validate current CA pair before promotion: %w", err)
	}
	newCA, newCAKey, err := loadNamedCA(path(CANewCertFile), path(CANewKeyFile))
	if err != nil {
		return fmt.Errorf("load new CA pair: %w", err)
	}
	if err := validateCAKeyPair(newCA, newCAKey); err != nil {
		return fmt.Errorf("validate new CA pair: %w", err)
	}
	if err := newCA.CheckSignatureFrom(newCA); err != nil {
		return fmt.Errorf("new CA certificate is not self-signed: %w", err)
	}
	if serialHex(newCA.SerialNumber) != rollover.NewSerial {
		return fmt.Errorf("new CA does not match rollover state")
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
		clientPath := filepath.Join(clientsDir, entry.Name(), ClientCertFile)
		clientCert, err := parseCertFile(clientPath)
		if err != nil {
			if os.IsNotExist(err) {
				backupPath := clientPath + ".rollover-old"
				if _, backupErr := os.Lstat(backupPath); backupErr == nil {
					return fmt.Errorf("CA rollover is not complete: client %q has a previous certificate backup but no active certificate", entry.Name())
				} else if !os.IsNotExist(backupErr) {
					return fmt.Errorf("inspect client certificate backup for %q: %w", entry.Name(), backupErr)
				}
				continue
			}
			return fmt.Errorf("inspect client certificate %q before promotion: %w", entry.Name(), err)
		}
		if clientCert.Issuer.String() == oldCA.Subject.String() {
			return fmt.Errorf("CA rollover is not complete: active client certificate for %q still uses the old CA", entry.Name())
		}
		if clientCert.Issuer.String() == newCA.Subject.String() {
			state, ok := rollover.Issued[entry.Name()]
			if !ok || state.NewSerial != serialHex(clientCert.SerialNumber) || state.ConfirmedAt == 0 {
				return fmt.Errorf("CA rollover is not complete: new-CA client certificate for %q has no matching confirmed journal entry", entry.Name())
			}
			roots := x509.NewCertPool()
			roots.AddCert(newCA)
			if _, err := clientCert.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
				return fmt.Errorf("new-CA client certificate for %q does not verify: %w", entry.Name(), err)
			}
		} else {
			return fmt.Errorf("client certificate for %q has an unrecognized issuer", entry.Name())
		}
		backupPath := filepath.Join(clientsDir, entry.Name(), ClientCertFile+".rollover-old")
		if _, err := os.Lstat(backupPath); err == nil {
			backupCert, err := parseCertFile(backupPath)
			if err != nil {
				return fmt.Errorf("inspect previous client certificate backup for %q: %w", entry.Name(), err)
			}
			if backupCert.Issuer.String() != oldCA.Subject.String() {
				return fmt.Errorf("previous client certificate backup for %q has an unrecognized issuer", entry.Name())
			}
			oldRoots := x509.NewCertPool()
			oldRoots.AddCert(oldCA)
			if _, err := backupCert.Verify(x509.VerifyOptions{Roots: oldRoots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
				return fmt.Errorf("previous client certificate backup for %q does not verify: %w", entry.Name(), err)
			}
			state, ok := rollover.Issued[entry.Name()]
			if !ok || state.OldSerial != serialHex(backupCert.SerialNumber) || state.ConfirmedAt == 0 {
				return fmt.Errorf("previous client certificate backup for %q is not confirmed in the rollover journal", entry.Name())
			}
			revoked, err := IsRevoked(backupCert.SerialNumber)
			if err != nil {
				return fmt.Errorf("check previous client certificate revocation for %q: %w", entry.Name(), err)
			}
			if !revoked {
				return fmt.Errorf("previous client certificate for %q is not revoked", entry.Name())
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect previous client certificate backup for %q: %w", entry.Name(), err)
		}
	}
	if err := tls.LoadX509KeyPair(path(ServerCertFile), path(ServerKeyFile)); err != nil {
		return fmt.Errorf("validate current server certificate/key before promotion: %w", err)
	}
	for _, name := range rolloverBackupFiles {
		if err := os.Remove(path(name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale rollover backup %s: %w", name, err)
		}
	}
	for _, pair := range [][2]string{
		{CACertFile, rolloverBackupCACert},
		{CAKeyFile, rolloverBackupCAKey},
		{ServerCertFile, rolloverBackupServerCert},
		{ServerKeyFile, rolloverBackupServerKey},
		{ClientCertFile, rolloverBackupClientCert},
		{ClientKeyFile, rolloverBackupClientKey},
	} {
		if err := copyDurable(path(pair[0]), path(pair[1])); err != nil {
			return fmt.Errorf("backup %s before CA promotion: %w", pair[0], err)
		}
	}
	rollover.Phase = "promoting"
	if err := saveRollover(rollover); err != nil {
		persisted, loadErr := loadRollover()
		if loadErr != nil || persisted.Phase != "promoting" {
			return fmt.Errorf("persist CA promotion intent: %w", err)
		}
	}
	rollback := func(cause error) error {
		if restoreErr := restoreCARolloverBackups(); restoreErr != nil {
			return fmt.Errorf("%v; CA promotion rollback failed: %w", cause, restoreErr)
		}
		rollover.Phase = "dual"
		if saveErr := saveRollover(rollover); saveErr != nil {
			return fmt.Errorf("%v; rollback restored files but journal update failed: %w", cause, saveErr)
		}
		return cause
	}
	newCertPEM, err := os.ReadFile(path(CANewCertFile))
	if err != nil {
		return rollback(fmt.Errorf("read new CA certificate: %w", err))
	}
	newKeyPEM, err := os.ReadFile(path(CANewKeyFile))
	if err != nil {
		return rollback(fmt.Errorf("read new CA key: %w", err))
	}
	if err := writeDurable(path(CACertFile), newCertPEM, 0o644); err != nil {
		return rollback(fmt.Errorf("promote new CA certificate: %w", err))
	}
	if err := writeDurable(path(CAKeyFile), newKeyPEM, 0o600); err != nil {
		return rollback(fmt.Errorf("promote new CA key: %w", err))
	}
	for _, name := range []string{ServerCertFile, ServerKeyFile, ClientCertFile, ClientKeyFile} {
		if err := os.Remove(path(name)); err != nil && !os.IsNotExist(err) {
			return rollback(fmt.Errorf("remove old %s before re-issuance: %w", name, err))
		}
	}
	pub := ""
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_ip")); err == nil {
		pub = strings.TrimSpace(string(b))
	}
	if err := ensureServer(pub); err != nil {
		return rollback(fmt.Errorf("issue server certificate from new CA: %w", err))
	}
	if err := ensureClient(); err != nil {
		return rollback(fmt.Errorf("issue default client certificate from new CA: %w", err))
	}
	serverPair, err := tls.LoadX509KeyPair(path(ServerCertFile), path(ServerKeyFile))
	if err != nil {
		return rollback(fmt.Errorf("validate promoted server certificate/key: %w", err))
	}
	serverLeaf, err := x509.ParseCertificate(serverPair.Certificate[0])
	if err != nil {
		return rollback(fmt.Errorf("parse promoted server certificate: %w", err))
	}
	roots := x509.NewCertPool()
	roots.AddCert(newCA)
	serverVerify := x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if hostname := publicHostname(); hostname != "" {
		serverVerify.DNSName = hostname
	}
	if _, err := serverLeaf.Verify(serverVerify); err != nil {
		return rollback(fmt.Errorf("verify promoted server certificate against new CA: %w", err))
	}
	clientPair, err := tls.LoadX509KeyPair(path(ClientCertFile), path(ClientKeyFile))
	if err != nil {
		return rollback(fmt.Errorf("validate promoted default client certificate/key: %w", err))
	}
	clientLeaf, err := x509.ParseCertificate(clientPair.Certificate[0])
	if err != nil {
		return rollback(fmt.Errorf("parse promoted default client certificate: %w", err))
	}
	if _, err := clientLeaf.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return rollback(fmt.Errorf("verify promoted default client certificate against new CA: %w", err))
	}
	rollover.Phase = "done"
	if err := saveRollover(rollover); err != nil {
		// Do not guess whether the final journal rename became visible. The
		// promotion backups remain intact; startup recovery will either roll
		// back a promoting phase or finish cleanup for a done phase.
		return fmt.Errorf("CA promotion completed but final journal state is uncertain; backups retained: %w", err)
	}
	if err := cleanupCARolloverBackups(); err != nil {
		return fmt.Errorf("CA promoted but backup cleanup is pending: %w", err)
	}
	return nil
}

const (
	rolloverBackupCACert     = "ca-rollover-backup-ca.crt"
	rolloverBackupCAKey      = "ca-rollover-backup-ca.key"
	rolloverBackupServerCert = "ca-rollover-backup-server.crt"
	rolloverBackupServerKey  = "ca-rollover-backup-server.key"
	rolloverBackupClientCert = "ca-rollover-backup-client.crt"
	rolloverBackupClientKey  = "ca-rollover-backup-client.key"
)

var rolloverBackupFiles = []string{
	rolloverBackupCACert, rolloverBackupCAKey, rolloverBackupServerCert, rolloverBackupServerKey,
	rolloverBackupClientCert, rolloverBackupClientKey,
}

func validateCAKeyPair(cert *x509.Certificate, key *ecdsa.PrivateKey) error {
	if cert == nil || key == nil {
		return fmt.Errorf("CA certificate or key is nil")
	}
	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok || pub.X == nil || pub.Y == nil || key.PublicKey.X == nil || key.PublicKey.Y == nil {
		return fmt.Errorf("CA key is not an ECDSA P-256 key")
	}
	if pub.X.Cmp(key.PublicKey.X) != 0 || pub.Y.Cmp(key.PublicKey.Y) != 0 {
		return fmt.Errorf("CA certificate and private key do not match")
	}
	return nil
}

func writeDurable(dst string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".ca-rollover-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	n, err := tmp.Write(data)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if n != len(data) {
		_ = tmp.Close()
		return fmt.Errorf("short write: %d of %d bytes", n, len(data))
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	df, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer df.Close()
	return df.Sync()
}

func copyDurable(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular rollover source file: %s", src)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeDurable(dst, data, info.Mode().Perm())
}

func restoreCARolloverBackups() error {
	for _, pair := range [][2]string{
		{rolloverBackupCACert, CACertFile},
		{rolloverBackupCAKey, CAKeyFile},
		{rolloverBackupServerCert, ServerCertFile},
		{rolloverBackupServerKey, ServerKeyFile},
	} {
		if err := copyDurable(path(pair[0]), path(pair[1])); err != nil {
			return fmt.Errorf("restore %s: %w", pair[1], err)
		}
	}
	return nil
}

func cleanupCARolloverBackups() error {
	for _, name := range append(append([]string{}, rolloverBackupFiles...), CANewCertFile, CANewKeyFile) {
		if err := os.Remove(path(name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	clientsDir := filepath.Join(Dir(), "clients")
	clients, err := os.ReadDir(clientsDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("scan client rollover backup directories: %w", err)
	}
	for _, entry := range clients {
		if !entry.IsDir() {
			continue
		}
		clientDir := filepath.Join(clientsDir, entry.Name())
		for _, clientFile := range []string{
			ClientCertFile + ".rollover-old", ClientKeyFile + ".rollover-old",
			ClientCertFile + ".rollover-new", ClientKeyFile + ".rollover-new",
		} {
			if err := os.Remove(filepath.Join(clientDir, clientFile)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove client rollover file %s/%s: %w", entry.Name(), clientFile, err)
			}
		}
		clientDirFile, err := os.Open(clientDir)
		if err != nil {
			return fmt.Errorf("open client directory %s for sync: %w", entry.Name(), err)
		}
		if err := clientDirFile.Sync(); err != nil {
			_ = clientDirFile.Close()
			return fmt.Errorf("sync client directory %s: %w", entry.Name(), err)
		}
		if err := clientDirFile.Close(); err != nil {
			return fmt.Errorf("close client directory %s: %w", entry.Name(), err)
		}
	}
	dir, err := os.Open(Dir())
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// recoverCARolloverPromotion always rolls back an interrupted promotion.
// A retry can then run the complete promotion transaction again.
func recoverCARolloverPromotion() error {
	rollover, err := loadRollover()
	if err != nil {
		return err
	}
	switch rollover.Phase {
	case "promoting":
		if err := restoreCARolloverBackups(); err != nil {
			return fmt.Errorf("recover interrupted CA promotion: %w", err)
		}
		restoredCA, restoredKey, err := loadNamedCA(path(CACertFile), path(CAKeyFile))
		if err != nil {
			return fmt.Errorf("validate restored CA pair: %w", err)
		}
		if err := validateCAKeyPair(restoredCA, restoredKey); err != nil {
			return fmt.Errorf("validate restored CA pair: %w", err)
		}
		if _, err := tls.LoadX509KeyPair(path(ServerCertFile), path(ServerKeyFile)); err != nil {
			return fmt.Errorf("validate restored server certificate/key: %w", err)
		}
		if _, err := tls.LoadX509KeyPair(path(ClientCertFile), path(ClientKeyFile)); err != nil {
			return fmt.Errorf("validate restored default client certificate/key: %w", err)
		}
		rollover.Phase = "dual"
		if err := saveRollover(rollover); err != nil {
			return fmt.Errorf("persist CA promotion recovery: %w", err)
		}
	case "done":
		if err := cleanupCARolloverBackups(); err != nil {
			return fmt.Errorf("finish CA rollover cleanup: %w", err)
		}
	}
	return nil
}

// CARolloverAbort removes ca-new only when no client certificate has been
// issued by it. Once a client has received a new-CA identity, deleting that CA
// would make the identity unverifiable and can strand the Secondary.
func CARolloverAbort() error {
	rolloverMu.Lock()
	defer rolloverMu.Unlock()
	if err := recoverCARolloverPromotion(); err != nil {
		return fmt.Errorf("recover prior CA promotion: %w", err)
	}

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
			clientPath := filepath.Join(root, entry.Name(), ClientCertFile)
			clientCert, err := parseCertFile(clientPath)
			if err != nil {
				if os.IsNotExist(err) {
					if _, backupErr := os.Lstat(clientPath + ".rollover-old"); backupErr == nil {
						return fmt.Errorf("cannot abort CA rollover: client %q has an interrupted certificate replacement", entry.Name())
					} else if !os.IsNotExist(backupErr) {
						return fmt.Errorf("inspect client backup for %q before abort: %w", entry.Name(), backupErr)
					}
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
