package secondary

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signer.PublicKey()
}

func TestTrustHostKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	key := testKey(t)
	fp := FingerprintSHA256(key)
	if fp == "" || fp[:7] != "SHA256:" {
		t.Fatalf("fp %q", fp)
	}
	if err := TrustHostKey("203.0.113.50", "52222", key); err != nil {
		t.Fatal(err)
	}
	ok, err := HasTOFUPin("203.0.113.50")
	if err != nil || !ok {
		t.Fatalf("has pin %v %v", ok, err)
	}
	// same key again ok
	if err := TrustHostKey("203.0.113.50", "52222", key); err != nil {
		t.Fatal(err)
	}
	other := testKey(t)
	if err := TrustHostKey("203.0.113.50", "52222", other); err == nil {
		t.Fatal("expected mismatch refuse")
	}
	line, err := OpenSSHKnownHostsLine("203.0.113.50", "52222")
	if err != nil || line == "" {
		t.Fatalf("line %q %v", line, err)
	}
	// pin file material
	raw := base64.StdEncoding.EncodeToString(key.Marshal())
	if raw == "" {
		t.Fatal("empty")
	}
	_ = filepath.Join(dir, "secondary")
	if _, err := os.Stat(filepath.Join(dir, "secondary", "ssh_known_hosts.json")); err != nil {
		t.Fatal(err)
	}
}

func TestTrustHostKeyPinLost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	key := testKey(t)
	if err := TrustHostKey("203.0.113.51", "22", key); err != nil {
		t.Fatal(err)
	}
	// wipe pins but keep seen
	pkhMu.Lock()
	_ = saveRelayKH(map[string]string{})
	pkhMu.Unlock()
	other := testKey(t)
	if err := TrustHostKey("203.0.113.51", "22", other); err != ErrPinLost {
		t.Fatalf("got %v want ErrPinLost", err)
	}
}
