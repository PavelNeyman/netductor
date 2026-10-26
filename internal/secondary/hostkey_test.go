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

	"github.com/PavelNeyman/netductor/internal/paths"
)

func withTOFUDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// SecondaryDir is under state root — set via env if supported, else write path
	// paths package uses fixed /var/lib/netductor or XDG — use WriteOpenSSH after planting file at relayKnownPath
	return dir
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
	// Plant TOFU next to SecondaryDir by writing through saveRelayKH after ensuring dir
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	_ = os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700)
	// Isolate: if SecondaryDir is system path, skip
	if strings.HasPrefix(relayKnownPath(), "/var/") {
		// try paths under temp by writing only if writable
		if err := os.WriteFile(relayKnownPath(), []byte("{}"), 0o600); err != nil {
			t.Skip("cannot write TOFU path in test env:", err)
		}
	}
	f := map[string]string{"203.0.113.50": b64}
	raw, _ := json.Marshal(f)
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = os.Remove(relayKnownPath()) })

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
	b, _ := os.ReadFile(dest)
	if !strings.Contains(string(b), "[203.0.113.50]:52222") {
		t.Fatalf("file=%q", b)
	}

	_, err = OpenSSHKnownHostsLine("203.0.113.99", "52222")
	if err == nil {
		t.Fatal("want missing pin error")
	}
}

func TestLoadRelayKHCorrupt(t *testing.T) {
	_ = os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700)
	if err := os.WriteFile(relayKnownPath(), []byte("{not-json"), 0o600); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = os.Remove(relayKnownPath()) })
	_, err := loadRelayKH()
	if err == nil {
		t.Fatal("want corrupt error")
	}
}

func TestOpenSSHKnownHostsPort22(t *testing.T) {
	pk := testSigner(t)
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	_ = os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700)
	raw, _ := json.Marshal(map[string]string{"203.0.113.51": b64})
	if err := os.WriteFile(relayKnownPath(), append(raw, 10), 0o600); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = os.Remove(relayKnownPath()) })
	line, err := OpenSSHKnownHostsLine("203.0.113.51", "22")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(line, "[") {
		t.Fatalf("port 22 should not use bracket form: %q", line)
	}
	_ = paths.SecondaryDir // silence unused if needed
}
