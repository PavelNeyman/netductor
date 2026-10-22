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

func TestNDG2ReorderChunks(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "p.bin")
	enc := filepath.Join(dir, "e.ndg2")
	// two full-ish chunks: payload > 1MiB so we get index 0 and 1
	payload := bytes.Repeat([]byte("ABCDEFGHIJKLMNOP"), (1<<20)/16+100)
	if err := os.WriteFile(plain, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	pass := "test-passphrase-32chars-minimum!!"
	if err := encryptFileStreamGCM(plain, enc, pass); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(enc)
	if err != nil {
		t.Fatal(err)
	}
	// Parse: magic(4) + size(4) + chunks...
	if len(raw) < 8 || string(raw[:4]) != "NDG2" {
		t.Fatal("bad magic")
	}
	off := 8
	type chunk struct{ start, end int }
	var chunks []chunk
	for off < len(raw) {
		start := off
		if off+12+4 > len(raw) {
			break
		}
		off += 12 // nonce
		ctLen := int(raw[off])<<24 | int(raw[off+1])<<16 | int(raw[off+2])<<8 | int(raw[off+3])
		off += 4
		if off+ctLen > len(raw) {
			t.Fatalf("bad len")
		}
		off += ctLen
		chunks = append(chunks, chunk{start, off})
	}
	if len(chunks) < 3 { // 2 data + end
		t.Fatalf("want >=3 chunks got %d", len(chunks))
	}
	// swap first two data chunks
	var out []byte
	out = append(out, raw[:8]...)
	out = append(out, raw[chunks[1].start:chunks[1].end]...)
	out = append(out, raw[chunks[0].start:chunks[0].end]...)
	for _, c := range chunks[2:] {
		out = append(out, raw[c.start:c.end]...)
	}
	bad := filepath.Join(dir, "reord")
	_ = os.WriteFile(bad, out, 0o600)
	if err := decryptFileStreamGCM(bad, filepath.Join(dir, "x"), pass); err == nil {
		t.Fatal("expected fail on reordered chunks")
	}
}
