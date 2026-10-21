package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNDG2RoundTrip(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "in.bin")
	payload := bytes.Repeat([]byte("netductor-backup-chunk-"), 80_000)
	if err := os.WriteFile(plain, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(dir, "out.ndenc")
	if err := encryptFileStreamGCM(plain, enc, "test-pass-key"); err != nil {
		t.Fatal(err)
	}
	dec := filepath.Join(dir, "dec.bin")
	if err := decryptFileStreamGCM(enc, dec, "test-pass-key"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dec)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("mismatch len got=%d want=%d", len(got), len(payload))
	}
	if err := decryptFileStreamGCM(enc, filepath.Join(dir, "bad.bin"), "wrong"); err == nil {
		t.Fatal("expected auth failure")
	}
}
