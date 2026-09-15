package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

const (
	DirName        = "mtls"
	CACertFile     = "ca.crt"
	CAKeyFile      = "ca.key"
	ServerCertFile = "server.crt"
	ServerKeyFile  = "server.key"
	ClientCertFile = "client.crt"
	ClientKeyFile  = "client.key"
	AgentTLSPort   = "8789"
)

func Dir() string { return filepath.Join(paths.EtcDir(), "secrets", DirName) }
func path(name string) string { return filepath.Join(Dir(), name) }

func EnsureAll(serverIP string) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	if err := ensureCA(); err != nil {
		return err
	}
	if err := ensureServer(serverIP); err != nil {
		return err
	}
	return ensureClient()
}

func ensureCA() error {
	if fileOK(path(CACertFile)) && fileOK(path(CAKeyFile)) {
		return nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(), Subject: pkix.Name{CommonName: "netductor-agent-ca", Organization: []string{"netductor"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true, MaxPathLen: 1,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := writeCert(path(CACertFile), der); err != nil {
		return err
	}
	return writeKey(path(CAKeyFile), key)
}

func ensureServer(serverIP string) error {
	if fileOK(path(ServerCertFile)) && fileOK(path(ServerKeyFile)) {
		return nil
	}
	caCert, caKey, err := loadCA()
	if err != nil {
		return err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(), Subject: pkix.Name{CommonName: "netductor-agent-server", Organization: []string{"netductor"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{"nd-primary", "localhost", "netductor-agent-server"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	if ip := net.ParseIP(serverIP); ip != nil {
		tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := writeCert(path(ServerCertFile), der); err != nil {
		return err
	}
	return writeKey(path(ServerKeyFile), key)
}

func ensureClient() error {
	if fileOK(path(ClientCertFile)) && fileOK(path(ClientKeyFile)) {
		return nil
	}
	caCert, caKey, err := loadCA()
	if err != nil {
		return err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(), Subject: pkix.Name{CommonName: "netductor-agent-client", Organization: []string{"netductor"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := writeCert(path(ClientCertFile), der); err != nil {
		return err
	}
	return writeKey(path(ClientKeyFile), key)
}

func loadCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(path(CACertFile))
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(path(CAKeyFile))
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	kblock, _ := pem.Decode(keyPEM)
	key, err := x509.ParseECPrivateKey(kblock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func ServerTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(path(ServerCertFile), path(ServerKeyFile))
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(path(CACertFile))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	return &tls.Config{Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert, MinVersion: tls.VersionTLS13}, nil
}

func ClientTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(path(ClientCertFile), path(ClientKeyFile))
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(path(CACertFile))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	return &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: pool, MinVersion: tls.VersionTLS13, ServerName: "netductor-agent-server"}, nil
}

func ServerReady() bool {
	return fileOK(path(CACertFile)) && fileOK(path(ServerCertFile)) && fileOK(path(ServerKeyFile))
}
func ClientReady() bool {
	return fileOK(path(CACertFile)) && fileOK(path(ClientCertFile)) && fileOK(path(ClientKeyFile))
}

func CopyClientMaterial() (ca, cert, key []byte, err error) {
	ca, err = os.ReadFile(path(CACertFile))
	if err != nil {
		return
	}
	cert, err = os.ReadFile(path(ClientCertFile))
	if err != nil {
		return
	}
	key, err = os.ReadFile(path(ClientKeyFile))
	return
}

func WriteClientMaterial(ca, cert, key []byte) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path(CACertFile), ca, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(path(ClientCertFile), cert, 0o600); err != nil {
		return err
	}
	return os.WriteFile(path(ClientKeyFile), key, 0o600)
}

func fileOK(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Size() > 0
}
func serial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return n
}
func writeCert(p string, der []byte) error {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der})
}
func writeKey(p string, key *ecdsa.PrivateKey) error {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}
