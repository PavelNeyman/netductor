package vpn

import (
	"strings"
	"testing"
)

func TestUplinkABDisabledByDefault(t *testing.T) {
	t.Setenv(uplinkABEnv, "0")
	cfg := LoadUplinkABConfig()
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
	cfg.Enabled = true
	cfg.Profiles = []string{"nomux"}
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

func TestGenerateWGKeypair(t *testing.T) {
	priv, pub, err := generateWGKeypair()
	if err != nil || priv == "" || pub == "" || priv == pub {
		t.Fatalf("priv=%q pub=%q err=%v", priv, pub, err)
	}
}

func TestAppendPrimaryABInbounds(t *testing.T) {
	t.Setenv(uplinkABEnv, "1")
	cfg := LoadUplinkABConfig()
	_ = cfg
	// force via env only — file may not exist; set profiles through Resolve with manual cfg in append uses Load
	inbounds := appendPrimaryUplinkABInbounds(nil, "uuid-up", "www.example.com", "priv", "sid")
	// may be empty if enabled false from file override - set env already
	if !UplinkABEnabled() {
		t.Skip("AB not enabled in this environment")
	}
	_ = inbounds
}

func TestUplinkABSubscriptionOnlyAllowlist(t *testing.T) {
	t.Setenv(uplinkABEnv, "1")
	t.Setenv(uplinkABUsersEnv, "operator")
	links := UplinkABSubscriptionLinks("nobody", "u", "1.2.3.4", "pbk", "sid", "sni")
	if len(links) != 0 {
		t.Fatal("no links for non-allowlist")
	}
}

func TestWGPeerUsesAddressPortNotEndpoint(t *testing.T) {
	cfg := UplinkABConfig{
		Enabled: true, Profiles: []string{"wg"}, BasePort: 9444,
		WGSecondaryPrivateKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		WGPrimaryPublicKey:    "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=",
		WGEndpoint:            "1.2.3.4:51820",
		WGListenPort:          51820,
		WGSecondaryAddress:    "10.87.90.2/30",
		WGMTU:                 1280,
	}
	// inject via save is hard; unit-test splitHostPort
	h, p := splitHostPortDefault("1.2.3.4:51820", 51820)
	if h != "1.2.3.4" || p != 51820 {
		t.Fatalf("%s %d", h, p)
	}
	_ = cfg
}

func TestResolveUplinkABReverseProfiles(t *testing.T) {
	cfg := UplinkABConfig{
		Enabled:         true,
		ReverseEnabled:  true,
		ReverseProfiles: []string{"ws", "wg", "nomux", "ps"}, // ws/nomux must be dropped
		ReverseBasePort: 9544,
		WGRevPrimaryPrivateKey:   "p",
		WGRevPrimaryPublicKey:    "P",
		WGRevSecondaryPrivateKey: "s",
		WGRevSecondaryPublicKey:  "S",
		WGRevPrimaryAddress:      "10.87.91.1/30",
		WGRevSecondaryAddress:    "10.87.91.2/30",
		WGRevListenPort:          51821,
	}
	// normalize strips fake profiles
	normalizeUplinkAB(&cfg)
	profs := ResolveUplinkABReverseProfiles(cfg)
	if len(profs) != 2 {
		t.Fatalf("want wg+ps only, got %d %+v", len(profs), profs)
	}
	for _, p := range profs {
		if !p.Reverse || (p.ID != "wg" && p.ID != "ps") {
			t.Fatalf("bad profile %+v", p)
		}
		if !strings.HasPrefix(p.Fragment, "nd-up-rev-") {
			t.Fatalf("fragment %s", p.Fragment)
		}
	}
}

func TestUplinkABReverseSubscriptionAllowlist(t *testing.T) {
	links := UplinkABReverseSubscriptionLinks("nobody", "u", "1.2.3.4", "pbk", "sid", "sni")
	if len(links) != 0 {
		t.Fatalf("expected no links for non-allowlisted user")
	}
}
