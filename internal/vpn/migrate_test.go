package vpn

import "testing"

func TestMigrateUserPolicies(t *testing.T) {
	r := &registry{Users: []UserRecord{
		{Name: "Alice"},
		{Name: "operator"},
		{Name: "TV", Kind: "device", SubProfile: "primary"},
	}}
	if !migrateUserPolicies(r) {
		t.Fatal("expected changes")
	}
	if r.Users[0].Kind != "user" {
		t.Fatalf("kind=%q", r.Users[0].Kind)
	}
	if r.Users[0].AllowInternet == nil || !*r.Users[0].AllowInternet {
		t.Fatal("allow_internet default")
	}
	if r.Users[0].ServicesMode != "list" {
		t.Fatal("services_mode")
	}
	if r.Users[0].SubProfile != SubSecondary {
		t.Fatalf("Alice sub=%q", r.Users[0].SubProfile)
	}
	if r.Users[1].SubProfile != SubBoth {
		t.Fatalf("operator sub=%q", r.Users[1].SubProfile)
	}
	if r.Users[2].Kind != "device" || r.Users[2].SubProfile != SubPrimary {
		t.Fatalf("device %+v", r.Users[2])
	}
	if migrateUserPolicies(r) {
		t.Fatal("second pass should be idempotent")
	}
}
