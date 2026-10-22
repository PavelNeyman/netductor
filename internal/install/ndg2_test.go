package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNDG2RoundTripAndTamper(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "p.bin")
	enc := filepath.Join(dir, "e.ndg2")
	out := filepath.Join(dir, "o.bin")
	payload := bytes.Repeat([]byte("netductor-backup-chunk-data-"), 100)
	if err := os.WriteFile(plain, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	pass := "test-passphrase-32chars-minimum!!"
	if err := encryptFileStreamGCM(plain, enc, pass); err != nil {
		t.Fatal(err)
	}
	if err := decryptFileStreamGCM(enc, out, pass); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if !bytes.Equal(got, payload) {
		t.Fatalf("roundtrip mismatch %d vs %d", len(got), len(payload))
	}
	// truncate end marker
	raw, _ := os.ReadFile(enc)
	if len(raw) < 40 {
		t.Fatal("enc too small")
	}
	trunc := filepath.Join(dir, "trunc")
	_ = os.WriteFile(trunc, raw[:len(raw)-20], 0o600)
	if err := decryptFileStreamGCM(trunc, filepath.Join(dir, "x"), pass); err == nil {
		t.Fatal("expected fail on truncated")
	}
	// forge end marker: append random bytes as fake marker won't auth
	forged := append(append([]byte{}, raw[:len(raw)-30]...), make([]byte, 40)...)
	_ = os.WriteFile(filepath.Join(dir, "forge"), forged, 0o600)
	if err := decryptFileStreamGCM(filepath.Join(dir, "forge"), filepath.Join(dir, "y"), pass); err == nil {
		t.Fatal("expected fail on forged tail")
	}
}
