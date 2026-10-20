package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListCachedUSBPackages(t *testing.T) {
	dir := t.TempDir()
	for _, n := range defaultUSBPkgNames {
		_ = os.WriteFile(filepath.Join(dir, n+"-1.apk"), []byte("x"), 0o644)
	}
	got, ok := listCachedUSBPackages(dir, ".apk", DefaultUSBPkgNames())
	if !ok || len(got) != len(defaultUSBPkgNames) {
		t.Fatalf("got=%v ok=%v", got, ok)
	}
}
