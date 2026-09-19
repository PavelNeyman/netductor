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
