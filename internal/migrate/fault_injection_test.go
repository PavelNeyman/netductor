package migrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// F: fault-injection around journal + runner (no live stack apply).

func TestFaultCorruptJournalFailClosed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	_ = os.MkdirAll(filepath.Join(root, "migrate"), 0o700)
	if err := os.WriteFile(journalPath(), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadJournal()
	if err == nil {
		t.Fatal("corrupt journal must error")
	}
	_, err = RunRegistry(context.Background(), DefaultRegistry)
	if err == nil {
		t.Fatal("RunRegistry must fail closed on corrupt journal")
	}
}

func TestFaultMidApplyLeavesFailedAndAllowsRetry(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	_ = os.MkdirAll(filepath.Join(root, "etc"), 0o755)

	calls := 0
	m := &countMig{id: "fault-mid", need: true, applyFn: func() error {
		calls++
		if calls == 1 {
			return context.Canceled // simulate kill / cancel mid-apply
		}
		return nil
	}}
	_, err := RunRegistry(context.Background(), []Migration{m})
	if err == nil {
		t.Fatal("expected first apply failure")
	}
	pub, err := LoadJournalPublic()
	if err != nil {
		t.Fatal(err)
	}
	if pub["fault-mid"].Status != StatusFailed {
		t.Fatalf("status=%s", pub["fault-mid"].Status)
	}
	// Without clear, Required may still be true but completed/failed blocks re-run of same id depending on runner logic
	if err := ClearFailedRecords(); err != nil {
		t.Fatal(err)
	}
	m.need = true
	res, err := RunRegistry(context.Background(), []Migration{m})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(res.Ran) != 1 {
		t.Fatalf("calls=%d ran=%v", calls, res.Ran)
	}
}

func TestFaultRunningStatusTreatedAsInterrupted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	_ = os.MkdirAll(filepath.Join(root, "migrate"), 0o700)
	j := journalFile{Schema: 1, Records: map[string]record{
		"stuck": {ID: "stuck", Status: StatusRunning},
	}}
	if err := durableWriteJournal(j); err != nil {
		t.Fatal(err)
	}
	// PostCheck should flag StatusRunning
	_ = os.MkdirAll(filepath.Join(root, "etc", "secrets"), 0o700)
	_ = os.MkdirAll(filepath.Join(root, "etc", "sessions"), 0o700)
	_ = os.MkdirAll(filepath.Join(root, "etc", "clients"), 0o700)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	// minimal critical dirs via ensure state - PostCheck needs all criticalDirs
	for _, d := range criticalDirs() {
		_ = os.MkdirAll(d, 0o700)
	}
	_ = os.WriteFile(filepath.Join(root, "etc", "role"), []byte("primary\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "installed_version"), []byte("0.9.653\n"), 0o644)
	err := PostCheck()
	if err == nil {
		t.Fatal("PostCheck must fail on StatusRunning")
	}
}

func TestFaultJournalAtomicReplace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	_ = os.MkdirAll(filepath.Join(root, "migrate"), 0o700)
	j := journalFile{Schema: 1, Records: map[string]record{
		"a": {ID: "a", Status: StatusCompleted},
	}}
	if err := durableWriteJournal(j); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(journalPath())
	if err != nil {
		t.Fatal(err)
	}
	var parsed journalFile
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Records["a"].Status != StatusCompleted {
		t.Fatal("round-trip")
	}
	// no leftover .tmp
	if _, err := os.Stat(journalPath() + ".tmp"); err == nil {
		t.Fatal("tmp journal left behind")
	}
}
