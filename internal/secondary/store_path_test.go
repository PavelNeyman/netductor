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
	currentAID, currentAToken, err := IssueToken("secondary-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Heartbeat(currentAToken, HeartbeatIn{}); err != nil {
		t.Fatal(err)
	}

	devices, err := ListE()
	if err != nil {
		t.Fatal(err)
	}
	foundSecond := false
	foundCurrentA := false
	for _, d := range devices {
		if d.ID == secondID && d.Token == secondToken && d.Name == "secondary-b" {
			foundSecond = true
		}
		if d.ID == firstID {
			t.Fatalf("replaced pending identity %q is still present", firstID)
		}
		if d.ID == currentAID && d.Token == currentAToken && d.Name == "secondary-a" {
			foundCurrentA = true
		}
	}
	if !foundSecond {
		t.Fatalf("pre-existing pending identity %q was removed while preparing another name", secondID)
	}
	if !foundCurrentA {
		t.Fatalf("joined replacement identity %q was unexpectedly removed", currentAID)
	}
}
