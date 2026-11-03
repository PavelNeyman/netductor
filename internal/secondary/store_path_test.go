package secondary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDevicesPathIsSecondary(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", tmp)
	id, tok, err := IssueToken("test")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || tok == "" {
		t.Fatal("empty")
	}
	p := filepath.Join(tmp, "secondary", "devices.json")
	if _, err := os.Stat(p); err != nil {
		t.Fatal("expected secondary/devices.json", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "relay")); !os.IsNotExist(err) {
		t.Fatal("must not create relay/")
	}
}

func TestIssueTokenPreservesOtherPendingDevice(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", tmp)

	firstID, _, err := IssueToken("secondary-a")
	if err != nil {
		t.Fatal(err)
	}
	secondID, secondToken, err := IssueToken("secondary-b")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := IssueToken("secondary-a"); err != nil {
		t.Fatal(err)
	}

	devices, err := ListE()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range devices {
		if d.ID == secondID && d.Token == secondToken && d.Name == "secondary-b" {
			found = true
		}
		if d.ID == firstID {
			t.Fatalf("same-name pending identity %q was not replaced", firstID)
		}
	}
	if !found {
		t.Fatalf("pre-existing pending identity %q was removed while preparing another name", secondID)
	}
}
