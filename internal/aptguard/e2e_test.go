package aptguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// End-to-end on a fake rootfs: policy → scan → findings → preview → apply → rescan.
func TestE2EQuarantineAndRescan(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	apt := filepath.Join(root, "etc/apt")
	_ = os.MkdirAll(filepath.Join(apt, "sources.list.d"), 0o755)
	_ = os.WriteFile(filepath.Join(apt, "sources.list"), []byte(
		"deb https://deb.debian.org/debian bookworm main contrib non-free-firmware\n"+
			"deb https://security.debian.org/debian-security bookworm-security main\n",
	), 0o644)
	evil := filepath.Join(apt, "sources.list.d", "malware.list")
	_ = os.WriteFile(evil, []byte("deb https://evil.example/debian bookworm main\n"), 0o644)

	pol, err := EnsurePolicy(state)
	if err != nil {
		t.Fatal(err)
	}
	rep1, err := ScanHost("e2e-host", pol, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveReport(state, rep1); err != nil {
		t.Fatal(err)
	}
	var hasUnknown bool
	for _, f := range rep1.Findings {
		if f.Code == "UNKNOWN_REPOSITORY" {
			hasUnknown = true
		}
	}
	if !hasUnknown {
		t.Fatalf("expected UNKNOWN_REPOSITORY: %+v", rep1.Findings)
	}

	pv, err := BuildPreview("e2e-host", root, pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Changes) == 0 {
		t.Fatal("expected quarantine change")
	}
	if pv.Blocked != "" {
		t.Fatalf("blocked on fake root: %s", pv.Blocked)
	}

	if _, err := ApplyPreview(state, root, pv, "yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(evil); !os.IsNotExist(err) {
		t.Fatal("malware.list should be quarantined")
	}
	if _, err := os.Stat(evil + ".nd-quarantine"); err != nil {
		t.Fatal("quarantine file missing")
	}

	rep3, err := ScanHost("e2e-host", pol, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep3.Findings {
		if f.Code == "UNKNOWN_REPOSITORY" && strings.Contains(f.SourceFile, "malware") {
			t.Fatalf("still unknown malware: %+v", f)
		}
	}
	got, err := LoadLatest(state, "e2e-host")
	if err != nil {
		t.Fatal(err)
	}
	if got.ServerID != "e2e-host" {
		t.Fatal(got.ServerID)
	}
	if Stale(got, 24*time.Hour) {
		t.Fatal("fresh report should not be stale")
	}
}
