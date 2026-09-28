package domain

import "testing"

func TestExpandNoInvent(t *testing.T) {
	c := Config{Base: "nd.example.com"}
	c.Expand()
	if c.Primary != "" || c.VPN != "" || c.RedirectBase != "" {
		t.Fatalf("must not invent hosts from base: %+v", c)
	}
	if c.Base != "nd.example.com" {
		t.Fatalf("base %q", c.Base)
	}
}

func TestExpandKeepsExplicit(t *testing.T) {
	c := Config{
		Base:         "nd.example.com",
		Primary:      "core.example.com",
		VPN:          "vpn.example.com",
		RedirectBase: "https://redir.example.com:8443",
	}
	c.Expand()
	if c.Primary != "core.example.com" || c.VPN != "vpn.example.com" {
		t.Fatalf("%+v", c)
	}
}
