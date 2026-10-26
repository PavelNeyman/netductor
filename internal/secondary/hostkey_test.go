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

func TestOpenSSHKnownHostsLineFormat(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pk := signer.PublicKey()
	b64 := base64.StdEncoding.EncodeToString(pk.Marshal())
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	pk2, err := ssh.ParsePublicKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	ak := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk2)))
	line := "[203.0.113.9]:52222 " + ak
	if !strings.HasPrefix(ak, "ssh-ed25519 ") {
		t.Fatalf("authorized key %q", ak)
	}
	dest := filepath.Join(t.TempDir(), "kh")
	if err := os.WriteFile(dest, append([]byte(line), 10), 0o600); err != nil {
		t.Fatal(err)
	}
}
