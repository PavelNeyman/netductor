package gitstore

import (
	"os"
	"time"
	"path/filepath"
	"testing"
)

func TestBuildHostEnabledFromRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(dir, "etc"))
	_ = os.MkdirAll(filepath.Join(dir, "etc"), 0o755)
	if BuildHostEnabled() {
		t.Fatal("expected disabled with empty registry and no config")
	}
	st := BuildHostState{ID: "bh1", Label: "b", SSHHost: "1.2.3.4", SSHPort: 52222, SSHUser: "root", Token: "t"}
	if err := SaveBuildHostState(st); err != nil {
		t.Fatal(err)
	}
	if BuildHostEnabled() {
		t.Fatal("stale offline registry must not enable ship")
	}
	st.LastSeen = time.Now().UTC().Format(time.RFC3339)
	if err := SaveBuildHostState(st); err != nil {
		t.Fatal(err)
	}
	if !BuildHostEnabled() {
		t.Fatal("expected enabled when registry has online host")
	}
}
