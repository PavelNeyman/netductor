package paths

import (
	"path/filepath"
	"testing"
)

func TestSecondaryDir(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	want := filepath.Join(StateDir(), "secondary")
	if SecondaryDir() != want {
		t.Fatalf("got %s want %s", SecondaryDir(), want)
	}
	if SecondaryDevicesFile() != filepath.Join(want, "devices.json") {
		t.Fatal(SecondaryDevicesFile())
	}
}
