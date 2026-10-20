package edge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidBearerDevice(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_EDGE_DIR", dir)
	_ = os.Setenv("NETDUCTOR_ETC", dir)
	t.Cleanup(func() {
		_ = os.Unsetenv("NETDUCTOR_EDGE_DIR")
		_ = os.Unsetenv("NETDUCTOR_ETC")
	})
	_ = os.MkdirAll(filepath.Join(dir, "secrets"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "edge_bootstrap_token"),
		[]byte("boot-token-at-least-32-characters-xx\n"), 0o600)

	Enroll(map[string]any{"device_id": "d1"})
	tok, err := Approve("d1")
	if err != nil || tok == "" {
		t.Fatal(err, tok)
	}
	if !ValidBearer("Bearer " + tok) {
		t.Fatal("device token should work")
	}
	if ValidBearer("Bearer not-a-real-token-xxxxxxxxxxxxxx") {
		t.Fatal()
	}
	// global edge token denied by default
	_ = os.WriteFile(filepath.Join(dir, "secrets", "edge_token"),
		[]byte("global-token-at-least-32-characters-yy\n"), 0o600)
	if ValidBearer("Bearer global-token-at-least-32-characters-yy") {
		t.Fatal("global should be off")
	}
}
