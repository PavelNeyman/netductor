package aptguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewQuarantineUnknown(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	evil := filepath.Join(d, "evil.list")
	_ = os.WriteFile(evil, []byte("deb https://evil.example/debian bookworm main\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "etc/apt/sources.list"), []byte("deb https://deb.debian.org/debian bookworm main\n"), 0o644)

	pv, err := BuildPreview("t", root, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range pv.Changes {
		if c.Action == "quarantine" && strings.Contains(c.Path, "evil.list") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want quarantine evil: %+v", pv.Changes)
	}

	state := t.TempDir()
	_, err = ApplyPreview(state, root, pv, "no")
	if err == nil {
		t.Fatal("need confirm")
	}
	rep, err := ApplyPreview(state, root, pv, "yes")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(evil); !os.IsNotExist(err) {
		t.Fatal("evil should be quarantined")
	}
	if _, err := os.Stat(evil + ".nd-quarantine"); err != nil {
		t.Fatal("quarantine file missing")
	}
	_ = rep
}

func TestApplyEmptyConfirm(t *testing.T) {
	_, err := ApplyPreview(t.TempDir(), t.TempDir(), Preview{}, "")
	if err == nil {
		t.Fatal("want error")
	}
}
