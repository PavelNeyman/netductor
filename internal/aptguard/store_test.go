package aptguard

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadAndDiff(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "state")
	p, err := EnsurePolicy(state)
	if err != nil {
		t.Fatal(err)
	}
	if p.Version < 1 {
		t.Fatal(p.Version)
	}
	rep := Report{
		ServerID: "local", Module: "apt-repository-guard", Status: SevWarn,
		CheckedAt: time.Now().UTC(), PolicyVersion: p.Version,
		Findings: []Finding{{Code: "UNKNOWN_REPOSITORY", Severity: SevWarn, Message: "x"}},
	}
	if err := SaveReport(state, rep); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLatest(state, "local")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != SevWarn || len(got.Findings) != 1 {
		t.Fatalf("%+v", got)
	}
	prev := got
	cur := got
	cur.Findings = append(cur.Findings, Finding{Code: "MISSING_REQUIRED", Severity: SevFail, Message: "y"})
	add, rem := DiffFindings(prev, cur)
	if len(add) != 1 || len(rem) != 0 {
		t.Fatalf("add=%v rem=%v", add, rem)
	}
}
