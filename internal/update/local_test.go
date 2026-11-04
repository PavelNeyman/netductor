package update

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestTryLocalNamedAsset(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_RELEASES_DIR", dir)
	tag := "v0.0.1"
	td := filepath.Join(dir, tag)
	_ = os.MkdirAll(td, 0o755)
	name := "netductor-linux-amd64"
	body := []byte("fake-bin")
	_ = os.WriteFile(filepath.Join(td, name), body, 0o755)
	sum := sha256.Sum256(body)
	hexSum := hex.EncodeToString(sum[:])

	dest := filepath.Join(t.TempDir(), "out")
	ok, err := tryLocalNamedAsset(tag, name, dest)
	if err == nil || ok {
		t.Fatal("expected error without SHA256SUMS")
	}
	_ = os.WriteFile(filepath.Join(td, "SHA256SUMS"), []byte(hexSum+"  "+name+"\n"), 0o644)
	ok, err = tryLocalNamedAsset(tag, name, dest)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "fake-bin" {
		t.Fatal(string(b))
	}
}

func TestVerifyLocalReleaseAssets(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_RELEASES_DIR", dir)
	tagDir := filepath.Join(dir, "v1.2.3")
	if err := os.MkdirAll(tagDir, 0o755); err != nil {
		t.Fatal(err)
	}
	names := []string{"netductor-linux-amd64", "netductor-tg-linux-amd64"}
	var manifest string
	for _, name := range names {
		body := []byte("verified-" + name)
		if err := os.WriteFile(filepath.Join(tagDir, name), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		manifest += hex.EncodeToString(sum[:]) + "  " + name + "\n"
	}
	if err := os.WriteFile(filepath.Join(tagDir, "SHA256SUMS"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLocalReleaseAssets("v1.2.3", names...); err != nil {
		t.Fatalf("valid mirror rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tagDir, names[1]), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLocalReleaseAssets("v1.2.3", names...); err == nil {
		t.Fatal("tampered asset accepted")
	}
	if err := VerifyLocalReleaseAssets("v1.2.3", names[0], "missing-asset"); err == nil {
		t.Fatal("missing asset accepted")
	}
}
