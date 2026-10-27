package secondary

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	cb := provisionHostKey("203.0.113.52")
	pk2 := testSigner(t)
	if err := cb("203.0.113.52", nil, pk2); err == nil {
		t.Fatal("want host key change rejection")
	}
}
