package aptguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrixieClassicSourcesListPasses(t *testing.T) {
	root := t.TempDir()
	apt := filepath.Join(root, "etc", "apt")
	if err := os.MkdirAll(apt, 0o755); err != nil {
		t.Fatal(err)
	}
	// Typical stock Trixie sources.list (main only, no signed-by).
	content := `# deb cdrom:[Debian GNU/Linux 13.2.0 _Trixie_ - Official amd64 NETINST]/]/
deb https://deb.debian.org/debian/ trixie main
deb http://security.debian.org/debian-security trixie-security main
deb https://deb.debian.org/debian/ trixie-updates main
`
	if err := os.WriteFile(filepath.Join(apt, "sources.list"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := ScanHost("nd-primary", DefaultPolicy(), root)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != SevOK && rep.Status != SevWarn {
		t.Fatalf("status=%s findings=%+v", rep.Status, rep.Findings)
	}
	for _, f := range rep.Findings {
		if f.Severity == SevFail {
			t.Fatalf("unexpected FAIL: %+v", f)
		}
	}
}
