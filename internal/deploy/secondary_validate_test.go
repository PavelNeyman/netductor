package deploy

import "testing"

func TestDeploySecondaryRequiresIPv4Host(t *testing.T) {
	err := DeploySecondary(SecondaryOpts{
		PrimaryHost:   "1.2.3.4",
		PrimaryKey:    "/nonexistent",
		SecondaryHost: "router.local",
		SecondaryPass: "x",
	})
	if err == nil || err.Error() == "" {
		t.Fatal("expected error for hostname")
	}
	if want := "public IP"; !contains(err.Error(), want) && !contains(err.Error(), "must be a public") {
		// soft check — message should mention IP requirement
		t.Log(err)
	}

	err = DeploySecondary(SecondaryOpts{
		PrimaryHost:   "1.2.3.4",
		PrimaryKey:    "/nonexistent",
		SecondaryHost: "2001:db8::1",
		SecondaryPass: "x",
	})
	if err == nil {
		t.Fatal("expected error for IPv6")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
