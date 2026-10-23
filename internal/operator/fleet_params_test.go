package operator

import "testing"

func TestNormalizeFleetParamsFlatWeb(t *testing.T) {
	p := map[string]any{
		"do_primary":         true,
		"do_secondary":       true,
		"primary_host":       "1.2.3.4",
		"primary_password":   "x",
		"secondary_host":     "5.6.7.8",
		"secondary_password": "y",
		"key":                "~/.ssh/netductor_primary",
		"sni":                "api.vk.me",
		"domain_base":        "nd.example",
	}
	f, err := NormalizeFleetParams(p)
	if err != nil {
		t.Fatal(err)
	}
	if !f.DoPrimary || !f.DoSecondary {
		t.Fatalf("flags: %+v", f)
	}
	if f.Primary.Host != "1.2.3.4" {
		t.Fatalf("primary host %q", f.Primary.Host)
	}
	if f.Primary.DomainBase != "nd.example" {
		t.Fatalf("domain %q", f.Primary.DomainBase)
	}
	if f.Secondary.SecondaryHost != "5.6.7.8" {
		t.Fatalf("sec host %q", f.Secondary.SecondaryHost)
	}
	if f.Secondary.PrimaryHost != "1.2.3.4" {
		t.Fatalf("sec primary host %q", f.Secondary.PrimaryHost)
	}
	if f.Secondary.PrimaryKey == "" || f.Primary.SSHPrivateKey == "" {
		t.Fatal("key empty")
	}
}

func TestNormalizeFleetSecondaryOnly(t *testing.T) {
	p := map[string]any{
		"do_secondary":       true,
		"primary_host":       "1.2.3.4",
		"primary_key":        "/tmp/k",
		"secondary_host":     "5.6.7.8",
		"secondary_password": "y",
	}
	f, err := NormalizeFleetParams(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.DoPrimary || !f.DoSecondary {
		t.Fatalf("flags %+v", f)
	}
	if f.Secondary.PrimaryHost != "1.2.3.4" || f.Secondary.PrimaryKey != "/tmp/k" {
		t.Fatalf("sec primary creds %+v", f.Secondary)
	}
}

func TestNormalizeFleetNothingSelected(t *testing.T) {
	_, err := NormalizeFleetParams(map[string]any{"do_primary": false})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPrimaryFromFlatParams(t *testing.T) {
	s := PrimaryFromFlatParams(map[string]any{
		"host": "9.9.9.9", "password": "p", "key": "/a/b", "domain_base": "x.y",
		"ssh_port": "52222", "le_email": "a@b.c",
	})
	if s.Host != "9.9.9.9" || s.SSHPrivateKey != "/a/b" || s.DomainBase != "x.y" || s.SSHPort != 52222 {
		t.Fatalf("%+v", s)
	}
	if s.DomainEmail != "a@b.c" {
		t.Fatalf("email %q", s.DomainEmail)
	}
}
