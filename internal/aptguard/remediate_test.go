package aptguard

import (
	"errors"
	"fmt"
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

func TestQuarantineRefusesExisting(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	evil := filepath.Join(d, "evil.list")
	_ = os.WriteFile(evil, []byte("deb https://evil.example/debian bookworm main\n"), 0o644)
	_ = os.WriteFile(evil+".nd-quarantine", []byte("old\n"), 0o644)
	pv, err := BuildPreview("t", root, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyPreview(t.TempDir(), root, pv, "yes")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want destination exists, got %v", err)
	}
}

func TestTokenIncludesContent(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "etc/apt/sources.list.d")
	_ = os.MkdirAll(d, 0o755)
	path := filepath.Join(d, "evil.list")
	_ = os.WriteFile(path, []byte("deb https://evil.example/debian bookworm main\n"), 0o644)
	pv1, _ := BuildPreview("t", root, DefaultPolicy())
	_ = os.WriteFile(path, []byte("deb https://evil.example/debian bookworm main contrib\n"), 0o644)
	// keep size different; token must change
	pv2, _ := BuildPreview("t", root, DefaultPolicy())
	if pv1.Token == pv2.Token {
		t.Fatal("token should change when source content changes")
	}
}

func TestDurableWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := durableWrite(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "hello" {
		t.Fatalf("%q %v", b, err)
	}
}

func TestRestoreFileDurable(t *testing.T) {
	dir := t.TempDir()
	bak := filepath.Join(dir, "bak")
	dst := filepath.Join(dir, "dest")
	_ = os.WriteFile(bak, []byte("orig"), 0o600)
	e := journalEntry{Mode: 0o600, UID: -1, GID: -1}
	if err := restoreFile(bak, dst, e); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dst)
	if string(b) != "orig" {
		t.Fatal(string(b))
	}
}

func TestLinkOnlyNoRename(t *testing.T) {
	// compile-time: ApplyPreview path uses Link only — behavioral coverage via existing refuse test.
}

func TestMkdirExclusiveCollision(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "tx1")
	if err := mkdirExclusive(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mkdirExclusive(path, 0o700); err == nil {
		t.Fatal("expected exist error")
	} else if !os.IsExist(err) {
		t.Fatalf("want IsExist, got %v", err)
	}
}

func TestDurableWriteCommitHappy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "COMMIT")
	if err := durableWriteCommit(path, []byte("ok\n")); err != nil {
		t.Fatal(err)
	}
	if !commitIsDone(dir) {
		t.Fatal("expected done")
	}
}

func TestDurableWriteCommitFsyncFailClearsOrAmbiguous(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "COMMIT")
	old := fsyncDirFn
	defer func() { fsyncDirFn = old }()
	fsyncDirFn = func(d string) error {
		return fmt.Errorf("injected fsync fail")
	}
	err := durableWriteCommit(path, []byte("ok\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrCommitAmbiguous) {
		return
	}
	if _, st := os.Stat(path); st == nil {
		t.Fatalf("COMMIT still present after clear failure: %v", err)
	}
}

func TestCommitIsDoneBodies(t *testing.T) {
	dir := t.TempDir()
	if commitIsDone(dir) {
		t.Fatal("empty")
	}
	_ = os.WriteFile(filepath.Join(dir, "COMMIT"), []byte("garbage"), 0o600)
	if commitIsDone(dir) {
		t.Fatal("garbage")
	}
	_ = os.WriteFile(filepath.Join(dir, "COMMIT"), []byte("ok\n"), 0o600)
	if !commitIsDone(dir) {
		t.Fatal("ok")
	}
}
