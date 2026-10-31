package aptguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseListAndPolicy(t *testing.T) {
	root := t.TempDir()
	apt := filepath.Join(root, "etc/apt")
	_ = os.MkdirAll(filepath.Join(apt, "sources.list.d"), 0o755)
	main := filepath.Join(apt, "sources.list")
	content := `deb https://deb.debian.org/debian bookworm main
# deb https://evil.example/debian bookworm main
deb https://unknown.example/debian bookworm main
`
	if err := os.WriteFile(main, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p := DefaultPolicy()
	rep, err := ScanHost("test", p, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Sources) < 2 {
		t.Fatalf("sources=%d", len(rep.Sources))
	}
	var unknown, disabled bool
	for _, f := range rep.Findings {
		if f.Code == "UNKNOWN_REPOSITORY" {
			unknown = true
		}
		if f.Code == "SOURCE_DISABLED" {
			disabled = true
		}
	}
	if !unknown {
		t.Fatal("want UNKNOWN_REPOSITORY")
	}
	if !disabled {
		t.Fatal("want SOURCE_DISABLED for commented line")
	}
}

func TestDeb822Parse(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	src := `Types: deb
URIs: https://deb.debian.org/debian
Suites: bookworm
Components: main
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg
`
	if err := os.WriteFile(filepath.Join(d, "debian.sources"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := ScanHost("t", DefaultPolicy(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Sources) != 1 || !rep.Sources[0].Enabled {
		t.Fatalf("%+v", rep.Sources)
	}
}
