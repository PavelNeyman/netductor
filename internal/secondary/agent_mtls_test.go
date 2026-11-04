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

func TestHeartbeatPersistsServerObservedTLSIdentity(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	id, token, err := IssueToken("secondary-test")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	d, _, err := HeartbeatWithTLSIdentity(token, HeartbeatIn{}, "ABCDEF123")
	if err != nil {
		t.Fatalf("HeartbeatWithTLSIdentity: %v", err)
	}
	if d.ID != id {
		t.Fatalf("heartbeat device ID = %q, want %q", d.ID, id)
	}
	if d.MTLSSerial != "abcdef123" || d.MTLSSeenAt.IsZero() {
		t.Fatalf("server-observed TLS identity not persisted: serial=%q seen_at=%v", d.MTLSSerial, d.MTLSSeenAt)
	}
}

func TestPublicIPPrefersPinnedDeploymentIdentity(t *testing.T) {
	root := t.TempDir()
	etcDir := filepath.Join(root, "etc")
	t.Setenv("NETDUCTOR_ETC", etcDir)
	t.Setenv("PUBLIC_IP", "203.0.113.99")
	if err := os.MkdirAll(filepath.Join(etcDir, "secrets"), 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(etcDir, "secrets", "public_ip"), []byte("198.51.100.20\n"), 0o600); err != nil { t.Fatal(err) }
	if got := publicIP(); got != "198.51.100.20" {
		t.Fatalf("publicIP() = %q, want pinned deployment address", got)
	}
}
