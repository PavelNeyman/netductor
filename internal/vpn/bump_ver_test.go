package vpn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestBumpSecondaryConfigVerRequiresRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	// no devices.json
	err := bumpSecondaryConfigVer()
	if err == nil {
		t.Fatal("expected error when registry missing")
	}
	sec := filepath.Join(dir, "secondary")
	if err := os.MkdirAll(sec, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sec, "devices.json")
	if err := os.WriteFile(path, []byte(`{"config_ver":3,"devices":[]}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := bumpSecondaryConfigVer(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		t.Fatal(string(b))
	}
	if int(m["config_ver"].(float64)) != 4 {
		t.Fatalf("got %v want 4", m["config_ver"])
	}
	_ = paths.StateDir() // keep import used if needed
}
