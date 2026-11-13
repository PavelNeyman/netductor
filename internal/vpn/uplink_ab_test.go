package vpn

import (
	"os"
	"testing"
)

func TestUplinkABDisabledByDefault(t *testing.T) {
	_ = os.Unsetenv(uplinkABEnv)
	cfg := LoadUplinkABConfig()
	// file may enable on real hosts; force env off
	t.Setenv(uplinkABEnv, "0")
	cfg = LoadUplinkABConfig()
	if cfg.Enabled {
		t.Fatal("expected disabled")
	}
	if len(ResolveUplinkABProfiles(cfg)) != 0 {
		t.Fatal("no profiles when disabled")
	}
}

func TestUplinkABNomuxWhenEnabled(t *testing.T) {
	t.Setenv(uplinkABEnv, "1")
	t.Setenv(uplinkABUsersEnv, "operator")
	cfg := LoadUplinkABConfig()
	cfg.Profiles = []string{"nomux"}
	cfg.Enabled = true
	profs := ResolveUplinkABProfiles(cfg)
	if len(profs) != 1 || profs[0].ID != "nomux" {
		t.Fatalf("got %#v", profs)
	}
	if !UplinkABUserAllowed("operator") {
		t.Fatal("operator allowed")
	}
	if UplinkABUserAllowed("family") {
		t.Fatal("family not allowed")
	}
}

func TestUplinkABSkipWGWithoutMaterial(t *testing.T) {
	cfg := UplinkABConfig{Enabled: true, Profiles: []string{"wg"}, BasePort: 9444}
	if n := ResolveUplinkABProfiles(cfg); len(n) != 0 {
		t.Fatalf("wg without keys must skip: %v", n)
	}
}
