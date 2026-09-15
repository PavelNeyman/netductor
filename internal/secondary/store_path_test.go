package secondary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMigratesLegacyRelayPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", tmp)
	legacy := filepath.Join(tmp, "relay")
	_ = os.MkdirAll(legacy, 0o700)
	raw, _ := json.Marshal(registry{ConfigVer: 7, Devices: []Device{{ID: "x", Name: "nd-secondary"}}})
	if err := os.WriteFile(filepath.Join(legacy, "devices.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := load()
	if err != nil {
		t.Fatal(err)
	}
	if r.ConfigVer != 7 || len(r.Devices) != 1 {
		t.Fatalf("migrate failed: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(tmp, "secondary", "devices.json")); err != nil {
		t.Fatal("expected secondary/devices.json after migrate")
	}
}
