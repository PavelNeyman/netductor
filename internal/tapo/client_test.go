package tapo

import (
	"testing"
)

func TestNonceLen(t *testing.T) {
	n := nonce8()
	if len(n) != 16 {
		t.Fatalf("want 16 hex chars, got %d %q", len(n), n)
	}
}

func TestPKCS7(t *testing.T) {
	b := pkcs7Pad([]byte("hello"), 16)
	out, err := pkcs7Unpad(b)
	if err != nil || string(out) != "hello" {
		t.Fatal(err, out)
	}
}

func TestAuthHashes(t *testing.T) {
	h1 := authHashV1("user", "pass")
	h2 := authHashV2("user", "pass")
	if len(h1) != 16 || len(h2) != 32 {
		t.Fatalf("lens %d %d", len(h1), len(h2))
	}
}

func TestKlapKeyDeriveLen(t *testing.T) {
	local := make([]byte, 16)
	remote := make([]byte, 16)
	auth := authHashV2("a", "b")
	key := sha256b(append(append(append([]byte("lsk"), local...), remote...), auth...))[:16]
	if len(key) != 16 {
		t.Fatal(len(key))
	}
}
