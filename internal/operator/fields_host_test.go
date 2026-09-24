package operator

import "testing"

func TestValidHost(t *testing.T) {
	ok := []string{"1.2.3.4", "example.com", "p.nd.neyman.top", "2001:db8::1"}
	for _, h := range ok {
		if !ValidHost(h) {
			t.Fatalf("expected ok: %q", h)
		}
	}
	bad := []string{"", "evil;rm", "host has space", "-oProxyCommand=x", "a$(id).com", "x|y", "host`id`"}
	for _, h := range bad {
		if ValidHost(h) {
			t.Fatalf("expected reject: %q", h)
		}
	}
}

func TestValidUser(t *testing.T) {
	if !ValidUser("root") || !ValidUser("netductor") {
		t.Fatal("expected ok users")
	}
	if ValidUser("root;id") || ValidUser("") || ValidUser("-bad") {
		t.Fatal("expected reject bad users")
	}
}
