package edgestorage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteOfflinePack(t *testing.T) {
	root := t.TempDir()
	// redirect Dir via writing into expected path won't work — LatestB uses fixed path.
	// Smoke: SaveManifest + file in Dir still needs /var/lib — skip if not writable
	id := "test-dev"
	dir := Dir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Skip(err)
	}
	defer os.RemoveAll(dir)
	name := "backup-B-test.tar.gz"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = SaveManifest(id, Manifest{Kind: "B", File: name})
	out, err := WriteOfflinePack(id, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "overlay-B.tar.gz")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "manifest.json")); err != nil {
		t.Fatal(err)
	}
}
