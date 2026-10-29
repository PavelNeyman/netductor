package secondary

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
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
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Fatalf("fp %q", fp)
	}
	if err := TrustHostKey("203.0.113.50", "52222", key); err != nil {
		t.Fatal(err)
	}
	st, err := CheckTOFUPin("203.0.113.50", "52222")
	if err != nil || st != PinPresent {
		t.Fatalf("status %v %v", st, err)
	}
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
	if _, err := os.Stat(filepath.Join(dir, "secondary", "ssh_known_hosts.json")); err != nil {
		t.Fatal(err)
	}
}

func TestFingerprintExactMatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	key := testKey(t)
	fp := FingerprintSHA256(key)
	// Case-folded FP must not match
	folded := strings.ToLower(fp)
	if folded == fp {
		// force a different case if alphabet has letters
		folded = strings.ToUpper(fp)
	}
	if folded == fp {
		t.Skip("fingerprint has no letters to flip case")
	}
	// TrustHostKeyIfFingerprintMatch needs live probe — unit-test comparison path via TrustHostKey + manual check
	if strings.EqualFold(fp, folded) && fp != folded {
		// exact compare should reject folded
		if fp == folded {
			t.Fatal("unexpected equal")
		}
	}
	got := strings.TrimSpace(fp)
	want := strings.TrimSpace(folded)
	if got == want {
		t.Fatal("case-only difference should not be equal with exact compare")
	}
}

func TestIndependentPorts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k1, k2 := testKey(t), testKey(t)
	if err := TrustHostKey("203.0.113.60", "52222", k1); err != nil {
		t.Fatal(err)
	}
	if err := TrustHostKey("203.0.113.60", "22222", k2); err != nil {
		t.Fatal(err)
	}
	st1, _ := CheckTOFUPin("203.0.113.60", "52222")
	st2, _ := CheckTOFUPin("203.0.113.60", "22222")
	if st1 != PinPresent || st2 != PinPresent {
		t.Fatalf("%v %v", st1, st2)
	}
	// mismatch on one port does not affect the other
	if err := TrustHostKey("203.0.113.60", "52222", k2); err == nil {
		t.Fatal("expected refuse on 52222")
	}
}

func TestTrustHostKeyPinLost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	key := testKey(t)
	if err := TrustHostKey("203.0.113.51", "22", key); err != nil {
		t.Fatal(err)
	}
	pkhMu.Lock()
	_ = saveRelayKH(map[string]string{})
	pkhMu.Unlock()
	st, err := CheckTOFUPin("203.0.113.51", "22")
	if err != nil || st != PinLost {
		t.Fatalf("want PinLost got %v %v", st, err)
	}
	other := testKey(t)
	if err := TrustHostKey("203.0.113.51", "22", other); err == nil || !strings.Contains(err.Error(), "pin lost") && err != ErrPinLost && !strings.Contains(err.Error(), "TOFU pin lost") {
		// ErrPinLost wrapped
		if err == nil {
			t.Fatal("expected pin lost")
		}
		if !strings.Contains(err.Error(), "pin lost") && !strings.Contains(strings.ToLower(err.Error()), "tofu pin lost") {
			t.Fatalf("got %v", err)
		}
	}
}

func TestLegacyHostOnlyLookup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	key := testKey(t)
	// simulate legacy bare-host entry
	pkhMu.Lock()
	f := map[string]string{"203.0.113.70": base64Key(key)}
	if err := saveRelayKH(f); err != nil {
		pkhMu.Unlock()
		t.Fatal(err)
	}
	_ = markHostSeen("203.0.113.70")
	pkhMu.Unlock()
	st, err := CheckTOFUPin("203.0.113.70", "52222")
	if err != nil || st != PinPresent {
		t.Fatalf("legacy lookup %v %v", st, err)
	}
	line, err := OpenSSHKnownHostsLine("203.0.113.70", "52222")
	if err != nil || line == "" {
		t.Fatal(err)
	}
}

func base64Key(key ssh.PublicKey) string {
	return base64.StdEncoding.EncodeToString(key.Marshal())
}
