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
