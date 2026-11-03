package secondary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/mtls"
)

func setupSecondaryMTLSTest(t *testing.T) (string, []byte, []byte, []byte) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	if err := mtls.EnsureAll("127.0.0.1"); err != nil {
		t.Fatalf("EnsureAll: %v", err)
	}
	ca, cert, key, err := mtls.EnsureClientFor("secondary-test")
	if err != nil {
		t.Fatalf("EnsureClientFor: %v", err)
	}
	return root, ca, cert, key
}

func TestValidateSecondaryMTLSMaterial(t *testing.T) {
	_, ca, cert, key := setupSecondaryMTLSTest(t)
	if err := validateSecondaryMTLSMaterial("secondary-test", string(ca), string(cert), string(key)); err != nil {
		t.Fatalf("valid material rejected: %v", err)
	}
	if err := validateSecondaryMTLSMaterial("another-secondary", string(ca), string(cert), string(key)); err == nil {
		t.Fatal("material with a different node identity was accepted")
	}
	if err := validateSecondaryMTLSMaterial("secondary-test", string(ca), string(cert), "not a private key"); err == nil {
		t.Fatal("certificate with a mismatched/invalid key was accepted")
	}
}

func TestInstallSecondaryMTLSMaterialPreservesOtherFiles(t *testing.T) {
	_, ca, cert, key := setupSecondaryMTLSTest(t)
	serverCert := filepath.Join(mtls.Dir(), mtls.ServerCertFile)
	if _, err := os.Stat(serverCert); err != nil {
		t.Fatalf("server certificate fixture missing: %v", err)
	}
	if err := installSecondaryMTLSMaterial(string(ca), string(cert), string(key)); err != nil {
		t.Fatalf("installSecondaryMTLSMaterial: %v", err)
	}
	if _, err := os.Stat(serverCert); err != nil {
		t.Fatalf("unrelated server certificate was lost during mTLS replacement: %v", err)
	}
	if _, err := mtls.ClientTLSConfig(); err != nil {
		t.Fatalf("installed client material is not loadable: %v", err)
	}
}
