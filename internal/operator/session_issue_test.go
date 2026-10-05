package operator

import "testing"

func TestSanitizeSessionHours(t *testing.T) {
	ok, err := sanitizeSessionHours("72")
	if err != nil || ok != "72" {
		t.Fatalf("%v %v", ok, err)
	}
	if _, err := sanitizeSessionHours("72;id"); err == nil {
		t.Fatal("expected reject")
	}
	if _, err := sanitizeSessionHours("$(id)"); err == nil {
		t.Fatal("expected reject")
	}
	if _, err := sanitizeSessionHours("0"); err == nil {
		t.Fatal("expected range")
	}
}

func TestSanitizeSSHTarget(t *testing.T) {
	u, h, err := sanitizeSSHTargetUserHost("root", "2.27.118.70")
	if err != nil || u != "root" || h != "2.27.118.70" {
		t.Fatal(u, h, err)
	}
	if _, _, err := sanitizeSSHTargetUserHost("root", "host;id"); err == nil {
		t.Fatal("expected reject")
	}
}
