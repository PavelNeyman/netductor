package policy

import "testing"

func TestServiceRouteRules(t *testing.T) {
	cat := DefaultCatalog()
	subjects := []Subject{
		{Name: "operator", Policy: AccessPolicy{AllowInternet: true, ServicesMode: "all"}},
		{Name: "appletv", Policy: AccessPolicy{AllowInternet: true, Services: []string{"lampac"}}},
		{Name: "locked", Policy: AccessPolicy{AllowInternet: false, Services: []string{}}},
	}
	rules := ServiceRouteRules(subjects, cat)
	if len(rules) < 3 {
		t.Fatalf("expected several rules, got %d", len(rules))
	}
	foundRejectNet := false
	foundLampacAllow := false
	for _, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if au, ok := m["auth_user"].([]string); ok {
			for _, n := range au {
				if n == "locked" && (m["action"] == "reject" || m["outbound"] == "block") {
					foundRejectNet = true
				}
				if n == "appletv" && m["port"] == 9118 {
					foundLampacAllow = true
				}
			}
		}
	}
	if !foundRejectNet {
		t.Fatal("expected reject for locked users")
	}
	if !foundLampacAllow {
		t.Fatal("expected lampac allow for appletv")
	}
}
