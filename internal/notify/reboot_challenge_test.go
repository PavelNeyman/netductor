package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecoyCodesIncludeRealAndUnique(t *testing.T) {
	real := "4242"
	got := DecoyCodes(real, 3)
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	seen := map[string]bool{}
	found := false
	for _, c := range got {
		if len(c) != 4 {
			t.Fatalf("bad code %q", c)
		}
		if seen[c] {
			t.Fatalf("dup %s", c)
		}
		seen[c] = true
		if c == real {
			found = true
		}
	}
	if !found {
		t.Fatal("real code missing")
	}
}

func TestRebootChallengeOneShot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	code, err := BeginRebootChallenge("primary")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 4 {
		t.Fatalf("code %q", code)
	}
	path := filepath.Join(dir, "reboot-challenge.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := ConsumeRebootChallenge("primary", code); err != nil {
		t.Fatal(err)
	}
	// second consume must fail
	if err := ConsumeRebootChallenge("primary", code); err == nil {
		t.Fatal("expected fail after one-shot")
	}
}

func TestRebootChallengeMismatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	code, err := BeginRebootChallenge("primary")
	if err != nil {
		t.Fatal(err)
	}
	if err := ConsumeRebootChallenge("primary", "0000"); err == nil {
		t.Fatal("wrong code should fail")
	}
	// still consume once with wrong target after new begin
	code2, err := BeginRebootChallenge("secondary-x")
	if err != nil {
		t.Fatal(err)
	}
	if err := ConsumeRebootChallenge("primary", code2); err == nil {
		t.Fatal("wrong target should fail")
	}
	_ = code
}
