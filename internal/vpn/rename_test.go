package vpn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestRenameKeepsUUID(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", filepath.Join(dir, "etc"))
	_ = os.Setenv("NETDUCTOR_STATE", filepath.Join(dir, "state"))
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	_ = EnsureDirs()
	if _, err := AddNative("alice", "t"); err != nil {
		// ApplyConfig may fail without sing-box — ignore if user created
	}
	r, err := loadRegistry()
	if err != nil || findUser(r, "alice") == nil {
		// AddNative failed hard
		if err != nil {
			t.Skip(err)
		}
		t.Skip("no alice")
	}
	uuid := findUser(r, "alice").UUID
	_ = RenameNative("alice", "bob")
	r, _ = loadRegistry()
	u := findUser(r, "bob")
	if u == nil || u.UUID != uuid {
		t.Fatalf("rename broke uuid: %+v", u)
	}
	if findUser(r, "alice") != nil {
		t.Fatal("old name still present")
	}
}
