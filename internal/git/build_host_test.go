package gitstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildHostConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_BUILD_HOST_CONFIG", filepath.Join(dir, "bh.json"))
	c := BuildHostConfig{Enabled: true, SSHHost: "1.2.3.4", SSHPort: 52222, SSHUser: "root", Label: "b1"}
	if err := SaveBuildHost(c); err != nil {
		t.Fatal(err)
	}
	got := LoadBuildHost()
	if !got.Enabled || got.SSHHost != "1.2.3.4" || got.SSHPort != 52222 {
		t.Fatalf("%+v", got)
	}
	if BuildHostStatusLine() == "" {
		t.Fatal("status")
	}
	_ = os.WriteFile(filepath.Join(dir, "bh.json"), []byte(`{}`), 0o600)
}
