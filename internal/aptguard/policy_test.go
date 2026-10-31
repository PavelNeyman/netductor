package aptguard

import "testing"

func TestDefaultPolicyValid(t *testing.T) {
	p := DefaultPolicy()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyRejectsEmptyURI(t *testing.T) {
	p := DefaultPolicy()
	p.Repos[0].URIs = []string{""}
	if err := p.Validate(); err == nil {
		t.Fatal("want error")
	}
}

func TestPolicyRejectsDuplicateID(t *testing.T) {
	p := DefaultPolicy()
	p.Repos = append(p.Repos, p.Repos[0])
	if err := p.Validate(); err == nil {
		t.Fatal("want duplicate error")
	}
}
