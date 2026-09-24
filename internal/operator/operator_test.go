package operator

import "testing"

func TestApplyDomainFlags(t *testing.T) {
	p := PrimarySpec{DomainBase: "nd.example.com", DomainEmail: "a@b.c"}
	ApplyDomainFlags(&p)
	if !p.DomainLE || p.DomainHTTP {
		t.Fatalf("le=%v http=%v", p.DomainLE, p.DomainHTTP)
	}
	p = PrimarySpec{DomainBase: "nd.example.com"}
	ApplyDomainFlags(&p)
	if p.DomainLE || !p.DomainHTTP {
		t.Fatalf("le=%v http=%v", p.DomainLE, p.DomainHTTP)
	}
}

func TestPrimaryFromFields(t *testing.T) {
	m := map[string]string{
		"host": "1.2.3.4", "domain_base": "nd.example.com", "le_email": "x@y.z",
		"with_lampac": "yes", "gen_key": "yes",
	}
	get := func(k string) string { return m[k] }
	s := PrimaryFromFields(get)
	if s.Host != "1.2.3.4" || !s.DomainLE || !s.WithLampac {
		t.Fatalf("%+v", s)
	}
}

func TestFleetFromFields(t *testing.T) {
	m := map[string]string{
		"do_primary": "yes", "do_secondary": "no", "host": "1.1.1.1",
	}
	f := FleetFromFields(func(k string) string { return m[k] })
	if !f.DoPrimary || f.DoSecondary || f.Primary.Host != "1.1.1.1" {
		t.Fatalf("%+v", f)
	}
}
