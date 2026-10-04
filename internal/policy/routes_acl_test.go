package policy

import "testing"

func TestOverrideUsesEndpointAddr(t *testing.T) {
	cat := &Catalog{Version: 1, Services: []Service{{
		ID: "git", Kind: KindInternal,
		Endpoints: []Endpoint{{Addr: "10.1.2.3", Port: 2222, Proto: "tcp"}},
	}}}
	rules := ServiceRouteRules([]Subject{{Name: "Pavel", Policy: AccessPolicy{Services: []string{"git"}, ServicesMode: "list"}}}, cat)
	var saw bool
	for _, r := range rules {
		m := r.(map[string]any)
		if m["override_address"] == "10.1.2.3" && m["port"] == 2222 {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("expected dial addr from endpoint, got %#v", rules)
	}
}

func TestRelayUplinkNotAutoAllowed(t *testing.T) {
	rules := ServiceRouteRules([]Subject{{Name: "Pavel", Policy: AccessPolicy{ServicesMode: "all"}}}, DefaultCatalog())
	for _, r := range rules {
		m := r.(map[string]any)
		au, _ := m["auth_user"].([]string)
		for _, n := range au {
			if n == "relay-uplink" {
				t.Fatal("relay-uplink must not be auto-allowed")
			}
		}
	}
}

func TestDisabledServiceSkipped(t *testing.T) {
	ServiceEnabled = func(id string) bool { return id != "nvr" }
	defer func() { ServiceEnabled = nil }()
	rules := ServiceRouteRules([]Subject{{Name: "Pavel", Policy: AccessPolicy{ServicesMode: "all"}}}, DefaultCatalog())
	for _, r := range rules {
		m := r.(map[string]any)
		if m["port"] == 1984 {
			t.Fatal("nvr port must be absent while ServiceEnabled is false")
		}
	}
}
