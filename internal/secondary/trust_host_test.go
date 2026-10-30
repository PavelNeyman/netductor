package secondary

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
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
	fp := "SHA256:AbCdEfGhIjKlMnOpQrStUvWxYz0123456789+/=="
	if !fingerprintsMatch(fp, fp) {
		t.Fatal("identical")
	}
	if !fingerprintsMatch("  "+fp+"  ", fp) {
		t.Fatal("trim")
	}
	folded := strings.ToLower(fp)
	if folded == fp {
		folded = strings.ToUpper(fp)
	}
	if fingerprintsMatch(fp, folded) {
		t.Fatal("case-only difference must not match")
	}
	if fingerprintsMatch(fp, "SHA256:other") {
		t.Fatal("different must not match")
	}
}

func TestForgetSSHHostPortAware(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k1, k2 := testKey(t), testKey(t)
	if err := TrustHostKey("203.0.113.80", "52222", k1); err != nil {
		t.Fatal(err)
	}
	if err := TrustHostKey("203.0.113.80", "22222", k2); err != nil {
		t.Fatal(err)
	}
	// forget one port only
	if err := ForgetSSHHost("203.0.113.80|52222"); err != nil {
		t.Fatal(err)
	}
	st1, _ := CheckTOFUPin("203.0.113.80", "52222")
	st2, _ := CheckTOFUPin("203.0.113.80", "22222")
	if st1 == PinPresent {
		t.Fatal("52222 should be gone")
	}
	if st2 != PinPresent {
		t.Fatalf("22222 must remain, got %v", st2)
	}
	// bare host forget removes remaining
	if err := ForgetSSHHost("203.0.113.80"); err != nil {
		t.Fatal(err)
	}
	st2, _ = CheckTOFUPin("203.0.113.80", "22222")
	if st2 == PinPresent {
		t.Fatal("bare forget should clear all ports")
	}
}

func TestForgetSSHHostColonPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k := testKey(t)
	if err := TrustHostKey("203.0.113.81", "52222", k); err != nil {
		t.Fatal(err)
	}
	if err := ForgetSSHHost("203.0.113.81:52222"); err != nil {
		t.Fatal(err)
	}
	st, _ := CheckTOFUPin("203.0.113.81", "52222")
	if st == PinPresent {
		t.Fatal("colon form should forget endpoint")
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

func TestPortIndependenceAfterOpenSSHLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k := testKey(t)
	if err := TrustHostKey("203.0.113.90", "52222", k); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenSSHKnownHostsLine("203.0.113.90", "52222"); err != nil {
		t.Fatal(err)
	}
	// Other port must be Absent, not PinLost
	st, err := CheckTOFUPin("203.0.113.90", "22222")
	if err != nil {
		t.Fatal(err)
	}
	if st != PinAbsent {
		t.Fatalf("want PinAbsent for other port, got %v", st)
	}
	_, err = OpenSSHKnownHostsLine("203.0.113.90", "22222")
	if err == nil {
		t.Fatal("expected no pin on other port")
	}
	if strings.Contains(err.Error(), "pin lost") || strings.Contains(err.Error(), "TOFU pin lost") {
		t.Fatalf("other port must not be pin-lost: %v", err)
	}
}

func TestForgetEndpointRefusesWhenLegacyPresent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k1, k2 := testKey(t), testKey(t)
	pkhMu.Lock()
	f := map[string]string{
		"203.0.113.91":       base64Key(k1),
		"203.0.113.91|52222": base64Key(k2),
	}
	if err := saveRelayKH(f); err != nil {
		pkhMu.Unlock()
		t.Fatal(err)
	}
	_ = markHostSeen("203.0.113.91|52222")
	pkhMu.Unlock()
	err := ForgetSSHHost("203.0.113.91|52222")
	if err == nil || !strings.Contains(err.Error(), "legacy pin") {
		t.Fatalf("want refuse legacy, got %v", err)
	}
	st, _ := CheckTOFUPin("203.0.113.91", "52222")
	if st != PinPresent {
		t.Fatal("pins must remain after refused forget")
	}
	if err := ForgetSSHHost("203.0.113.91"); err != nil {
		t.Fatal(err)
	}
	st, _ = CheckTOFUPin("203.0.113.91", "52222")
	if st == PinPresent {
		t.Fatal("bare forget should remove all")
	}
}

func TestForgetEndpointOKWithoutLegacy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k1, k2 := testKey(t), testKey(t)
	if err := TrustHostKey("203.0.113.92", "52222", k1); err != nil {
		t.Fatal(err)
	}
	if err := TrustHostKey("203.0.113.92", "22222", k2); err != nil {
		t.Fatal(err)
	}
	if err := ForgetSSHHost("203.0.113.92|52222"); err != nil {
		t.Fatal(err)
	}
	st1, _ := CheckTOFUPin("203.0.113.92", "52222")
	st2, _ := CheckTOFUPin("203.0.113.92", "22222")
	if st1 == PinPresent {
		t.Fatal("52222 should be gone")
	}
	if st2 != PinPresent {
		t.Fatal("22222 must remain")
	}
}

func TestForgetHostClearsOrphanedSeen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	// Orphan: seen without pin
	pkhMu.Lock()
	if err := saveRelayKH(map[string]string{}); err != nil {
		pkhMu.Unlock()
		t.Fatal(err)
	}
	if err := markHostSeen("203.0.113.93|52222"); err != nil {
		pkhMu.Unlock()
		t.Fatal(err)
	}
	pkhMu.Unlock()
	st, err := CheckTOFUPin("203.0.113.93", "52222")
	if err != nil || st != PinLost {
		t.Fatalf("setup want PinLost got %v %v", st, err)
	}
	if err := ForgetSSHHost("203.0.113.93"); err != nil {
		t.Fatal(err)
	}
	st, err = CheckTOFUPin("203.0.113.93", "52222")
	if err != nil || st != PinAbsent {
		t.Fatalf("after forget want PinAbsent got %v %v", st, err)
	}
	// must not affect longer host names
	pkhMu.Lock()
	_ = markHostSeen("203.0.113.93.example|22")
	pkhMu.Unlock()
	if err := ForgetSSHHost("203.0.113.93"); err != nil {
		t.Fatal(err)
	}
	pkhMu.Lock()
	seen, _ := loadSeenHosts()
	pkhMu.Unlock()
	if !seen["203.0.113.93.example|22"] {
		t.Fatal("must not clear longer hostname prefix")
	}
}

func TestForgetHostSeenWriteFailThenRetry(t *testing.T) {
	// mutates package-level saveSeenHostsFn — do not parallelize this test
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	k := testKey(t)
	if err := TrustHostKey("203.0.113.94", "52222", k); err != nil {
		t.Fatal(err)
	}
	calls := 0
	prev := saveSeenHostsFn
	t.Cleanup(func() { saveSeenHostsFn = prev })
	saveSeenHostsFn = func(m map[string]bool) error {
		calls++
		if calls == 1 {
			return fmt.Errorf("simulated seen write fail")
		}
		return saveSeenHostsDisk(m)
	}
	err := ForgetSSHHost("203.0.113.94")
	if err == nil || !strings.Contains(err.Error(), "simulated seen write fail") {
		t.Fatalf("want seen write error, got %v", err)
	}
	// Pins may already be gone; seen may still mark PinLost
	st, _ := CheckTOFUPin("203.0.113.94", "52222")
	if st == PinPresent {
		t.Fatal("pin map should be cleared even if seen write failed")
	}
	// retry succeeds
	if err := ForgetSSHHost("203.0.113.94"); err != nil {
		t.Fatal(err)
	}
	st, err = CheckTOFUPin("203.0.113.94", "52222")
	if err != nil || st != PinAbsent {
		t.Fatalf("after retry want PinAbsent got %v %v", st, err)
	}
}
