package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func edgeTestMaterial(t *testing.T, nodeID string) (string, string, string) {
	t.Helper()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{CommonName: "edge-test-ca"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{CommonName: "netductor-agent-" + nodeID},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	clientPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return string(caPEM), string(clientPEM), string(keyPEM)
}

func TestValidateEdgeMTLSMaterial(t *testing.T) {
	ca, cert, key := edgeTestMaterial(t, "edge-1")
	if err := validateEdgeMTLSMaterial("edge-1", ca, cert, key); err != nil {
		t.Fatalf("valid edge material rejected: %v", err)
	}
	if err := validateEdgeMTLSMaterial("edge-2", ca, cert, key); err == nil {
		t.Fatal("certificate for another edge identity was accepted")
	}
	if err := validateEdgeMTLSMaterial("edge-1", ca, cert, "invalid key"); err == nil {
		t.Fatal("invalid client key was accepted")
	}
}

func TestInstallEdgeMTLSMaterialCanRestoreBothPaths(t *testing.T) {
	root := t.TempDir()
	oldDirs := mtlsDirs
	mtlsDirs = []string{
		filepath.Join(root, "agent", "mtls"),
		filepath.Join(root, "netductor", "secrets", "mtls"),
	}
	defer func() { mtlsDirs = oldDirs }()
	for _, dir := range mtlsDirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		for name, value := range map[string]string{
			"ca.crt": "old-ca", "client.crt": "old-cert", "client.key": "old-key", "extra.conf": "preserve-me",
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := installEdgeMTLSMaterial("new-ca", "new-cert", "new-key"); err != nil {
		t.Fatalf("installEdgeMTLSMaterial: %v", err)
	}
	for _, dir := range mtlsDirs {
		for name, want := range map[string]string{
			"ca.crt": "new-ca", "client.crt": "new-cert", "client.key": "new-key", "extra.conf": "preserve-me",
		} {
			got, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil || string(got) != want {
				t.Fatalf("%s/%s = %q, err=%v; want %q", dir, name, got, err, want)
			}
		}
	}
	if !restoreEdgeMTLSMaterial() {
		t.Fatal("restoreEdgeMTLSMaterial failed")
	}
	for _, dir := range mtlsDirs {
		for name, want := range map[string]string{
			"ca.crt": "old-ca", "client.crt": "old-cert", "client.key": "old-key", "extra.conf": "preserve-me",
		} {
			got, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil || string(got) != want {
				t.Fatalf("restored %s/%s = %q, err=%v; want %q", dir, name, got, err, want)
			}
		}
	}
}
