package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListCachedUSBPackages(t *testing.T) {
	dir := t.TempDir()
	for _, n := range usbPkgBaseNames {
		_ = os.WriteFile(filepath.Join(dir, n+"-1.apk"), []byte("x"), 0o644)
	}
	got, ok := listCachedUSBPackages(dir, ".apk")
	if !ok || len(got) != len(usbPkgBaseNames) {
		t.Fatalf("got=%v ok=%v", got, ok)
	}
}
