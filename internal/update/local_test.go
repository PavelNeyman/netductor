package update

import (
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
	// no SHA256SUMS → still ok
	dest := filepath.Join(t.TempDir(), "out")
	ok, err := tryLocalNamedAsset(tag, name, dest)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "fake-bin" {
		t.Fatal(string(b))
	}
}
