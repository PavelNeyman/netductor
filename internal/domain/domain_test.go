package domain

import "testing"

func TestExpand(t *testing.T) {
	c := Config{Base: "netductor.neyman.top", UseHTTPRedirect: true}
	c.Expand()
	if c.Primary != "primary.netductor.neyman.top" {
		t.Fatalf("primary %q", c.Primary)
	}
	if c.VPN != "vpn.netductor.neyman.top" {
		t.Fatalf("vpn %q", c.VPN)
	}
	if c.RedirectBase != "http://i.netductor.neyman.top" {
		t.Fatalf("redirect %q", c.RedirectBase)
	}
}
