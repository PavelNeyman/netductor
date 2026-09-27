package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOverride(t *testing.T) {
	_ = os.Setenv("NETDUCTOR_ETC", "/tmp/nd-test-etc")
	t.Cleanup(func() { _ = os.Unsetenv("NETDUCTOR_ETC") })
	if EtcDir() != "/tmp/nd-test-etc" {
		t.Fatal(EtcDir())
	}
}

func TestEnsureLayout(t *testing.T) {
	root := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	_ = os.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	_ = os.Setenv("NETDUCTOR_SHARE", filepath.Join(root, "share"))
	t.Cleanup(func() {
		_ = os.Unsetenv("NETDUCTOR_ETC")
		_ = os.Unsetenv("NETDUCTOR_STATE")
		_ = os.Unsetenv("NETDUCTOR_SHARE")
		_ = os.Unsetenv("NETDUCTOR_ROOT")
	})
	if err := EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{
		EtcDir(), filepath.Join(EtcDir(), "secrets"), SessionsDir(), ClientsDir(),
		StateDir(), MetricsDir(), EdgeDir(), AdminRoot(), ShareDir(),
	} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			t.Fatalf("%s: %v", d, err)
		}
	}
	if EdgeTokenFile() == "" || ProbesCfg() == "" || VPNBin() == "" {
		t.Fatal("empty paths")
	}
}
