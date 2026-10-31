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
	if pv.Token == "" {
		t.Fatal("token")
	}

	state := t.TempDir()
	_, err = ApplyPreview(state, root, pv, "no")
	if err == nil {
		t.Fatal("need confirm")
	}
	if _, err := ApplyPreview(state, root, pv, "yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(evil); !os.IsNotExist(err) {
		t.Fatal("evil should be quarantined")
	}
	if _, err := os.Stat(evil + ".nd-quarantine"); err != nil {
		t.Fatal("quarantine file missing")
	}
}

func TestMixedFileBlocksQuarantine(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	mixed := filepath.Join(d, "mixed.list")
	_ = os.WriteFile(mixed, []byte(
		"deb [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] https://deb.debian.org/debian bookworm main contrib non-free-firmware\n"+
			"deb https://evil.example/debian bookworm main\n",
	), 0o644)
	pv, err := BuildPreview("t", root, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if pv.Blocked == "" {
		t.Fatalf("want blocked mixed, changes=%v", pv.Changes)
	}
	for _, c := range pv.Changes {
		if c.Action == "quarantine" && strings.Contains(c.Path, "mixed") {
			t.Fatal("must not quarantine mixed file")
		}
	}
}

func TestApplyRejectsStaleToken(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	_ = os.WriteFile(filepath.Join(d, "evil.list"), []byte("deb https://evil.example/debian bookworm main\n"), 0o644)
	pv, err := BuildPreview("t", root, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	pv.Token = "deadbeef"
	_, err = ApplyPreview(t.TempDir(), root, pv, "yes")
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("want token error got %v", err)
	}
}

func TestApplyEmptyConfirm(t *testing.T) {
	_, err := ApplyPreview(t.TempDir(), t.TempDir(), Preview{}, "")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestDeb822EmptyTypesNoPanic(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	_ = os.WriteFile(filepath.Join(d, "bad.sources"), []byte("Types:\nURIs: https://deb.debian.org/debian\nSuites: bookworm\nComponents: main\n"), 0o644)
	_, err := ScanHost("t", DefaultPolicy(), root)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMatchPolicySuiteMissing(t *testing.T) {
	pol := DefaultPolicy()
	s := SourceEntry{Enabled: true, Type: "deb", URIs: []string{"https://deb.debian.org/debian"}, Suites: nil, Components: []string{"main"}}
	if matchPolicy(pol, s) != "mismatch" {
		t.Fatal(matchPolicy(pol, s))
	}
}

func TestParseListSignedBy(t *testing.T) {
	e, ok := parseDebLine("/t", `deb [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] https://deb.debian.org/debian bookworm main`, true)
	if !ok {
		t.Fatal("parse")
	}
	if e.SignedBy != "/usr/share/keyrings/debian-archive-keyring.gpg" {
		t.Fatal(e.SignedBy)
	}
}

func TestDeb822RedactURI(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	_ = os.WriteFile(filepath.Join(d, "x.sources"), []byte("Types: deb\nURIs: https://user:secret@evil.example/debian\nSuites: bookworm\nComponents: main\n"), 0o644)
	ents, _, err := parseDeb822File(filepath.Join(d, "x.sources"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 {
		t.Fatal(ents)
	}
	if strings.Contains(ents[0].URIs[0], "secret") {
		t.Fatal(ents[0].URIs[0])
	}
}
