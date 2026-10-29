package secondary

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func isolateState(t *testing.T) {
	t.Helper()
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
}

func testSigner(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s.PublicKey()
}

func TestOpenSSHKnownHostsLineFromStore(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"203.0.113.50": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Fatal(err)
	}

	line, err := OpenSSHKnownHostsLine("203.0.113.50", "52222")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "[203.0.113.50]:52222 ") {
		t.Fatalf("line=%q", line)
	}
	if !strings.Contains(line, "ssh-ed25519") {
		t.Fatalf("want ed25519 in %q", line)
	}

	dest := filepath.Join(t.TempDir(), "kh")
	if err := WriteOpenSSHKnownHostsFile("203.0.113.50", "52222", dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "[203.0.113.50]:52222") {
		t.Fatalf("file=%q", b)
	}

	_, err = OpenSSHKnownHostsLine("203.0.113.99", "52222")
	if err == nil {
		t.Fatal("want missing pin error")
	}
}

func TestLoadRelayKHCorrupt(t *testing.T) {
	isolateState(t)
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relayKnownPath(), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadRelayKH()
	if err == nil {
		t.Fatal("want corrupt error")
	}
}

func TestOpenSSHKnownHostsPort22(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"203.0.113.51": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	line, err := OpenSSHKnownHostsLine("203.0.113.51", "22")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(line, "[") {
		t.Fatalf("port 22 should not use bracket form: %q", line)
	}
}

func TestProvisionHostKeyRejectsChange(t *testing.T) {
	isolateState(t)
	pk1 := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk1.Marshal())
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"203.0.113.52": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	cb := provisionHostKey("203.0.113.52", 22)
	pk2 := testSigner(t)
	if err := cb("203.0.113.52", nil, pk2); err == nil {
		t.Fatal("want host key change rejection")
	}
}

func TestPinLostRejectsRelearnWithoutForget(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"203.0.113.60": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := markHostSeen("203.0.113.60"); err != nil {
		t.Fatal(err)
	}
	// Delete pin entry but keep seen
	if err := os.WriteFile(relayKnownPath(), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := OpenSSHKnownHostsLine("203.0.113.60", "52222")
	if err == nil || !errors.Is(err, ErrPinLost) {
		t.Fatalf("want ErrPinLost, got %v", err)
	}
}

func TestForgetAllowsFirstContactAgain(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	_ = os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700)
	raw, _ := json.Marshal(map[string]string{"203.0.113.61": b64})
	_ = os.WriteFile(relayKnownPath(), append(raw, 10), 0o600)
	_ = markHostSeen("203.0.113.61")
	if err := ForgetSSHHost("203.0.113.61"); err != nil {
		t.Fatal(err)
	}
	_, err := OpenSSHKnownHostsLine("203.0.113.61", "22")
	if err == nil {
		t.Fatal("want no pin after forget")
	}
	if errors.Is(err, ErrPinLost) {
		t.Fatal("after forget should be first-contact missing, not pin lost")
	}
}

func TestMigrateSeenFromLegacyPins(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	// Legacy: pins only, no seen file
	raw, _ := json.Marshal(map[string]string{"203.0.113.70": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(relaySeenPath()); !os.IsNotExist(err) {
		t.Fatal("seen should not exist yet")
	}
	if err := MigrateSSHTrustSeen(); err != nil {
		t.Fatal(err)
	}
	seen, err := loadSeenHosts()
	if err != nil {
		t.Fatal(err)
	}
	if !seen["203.0.113.70"] {
		t.Fatal("legacy pin must appear in seen after migration")
	}
	// Idempotent
	if err := MigrateSSHTrustSeen(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyPinThenLostStillPinLost(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	_ = os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700)
	raw, _ := json.Marshal(map[string]string{"203.0.113.71": b64})
	_ = os.WriteFile(relayKnownPath(), append(raw, 10), 0o600)
	// No seen file — migrate via OpenSSHKnownHostsLine path
	line, err := OpenSSHKnownHostsLine("203.0.113.71", "52222")
	if err != nil {
		t.Fatal(err)
	}
	if line == "" {
		t.Fatal("empty line")
	}
	// Drop pin, keep seen (migration wrote it)
	_ = os.WriteFile(relayKnownPath(), []byte("{}\n"), 0o600)
	_, err = OpenSSHKnownHostsLine("203.0.113.71", "52222")
	if !errors.Is(err, ErrPinLost) {
		t.Fatalf("want ErrPinLost after losing pin post-migration, got %v", err)
	}
}

func TestMatchingPinRestoresSeen(t *testing.T) {
	isolateState(t)
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	_ = os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700)
	raw, _ := json.Marshal(map[string]string{"203.0.113.72": b64})
	_ = os.WriteFile(relayKnownPath(), append(raw, 10), 0o600)
	// empty seen
	_ = os.WriteFile(relaySeenPath(), []byte("[]\n"), 0o600)
	cb := provisionHostKey("203.0.113.72", 22)
	if err := cb("203.0.113.72", nil, pk); err != nil {
		t.Fatal(err)
	}
	seen, err := loadSeenHosts()
	if err != nil {
		t.Fatal(err)
	}
	if !seen["203.0.113.72"] {
		t.Fatal("matching pin must restore seen marker")
	}
}

func TestProvisionHostKeyPinLostRefusesNewKey(t *testing.T) {
	isolateState(t)
	pkOld := testSigner(t)
	pkNew := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pkOld.Marshal())
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"203.0.113.73": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := markHostSeen("203.0.113.73"); err != nil {
		t.Fatal(err)
	}
	// Drop pin only
	if err := os.WriteFile(relayKnownPath(), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cb := provisionHostKey("203.0.113.73", 22)
	if err := cb("203.0.113.73", nil, pkNew); err == nil || !errors.Is(err, ErrPinLost) {
		t.Fatalf("want ErrPinLost, got %v", err)
	}
	// Must not have written new pin
	f, err := loadRelayKH()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f["203.0.113.73"]; ok {
		t.Fatal("must not re-TOFU after pin loss")
	}
}
