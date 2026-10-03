package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCatalogValidate(t *testing.T) {
	c := DefaultCatalog()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("lampac"); !ok {
		t.Fatal("lampac missing")
	}
	if _, ok := c.Get("internet"); !ok {
		t.Fatal("internet missing")
	}
}

func TestCatalogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	c := DefaultCatalog()
	if err := SaveCatalog(c); err != nil {
		t.Fatal(err)
	}
	path := CatalogPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	c2, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Services) != len(c.Services) {
		t.Fatalf("got %d services", len(c2.Services))
	}
}

func TestEnsureCatalogSeeds(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	c, err := EnsureCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Services) < 4 {
		t.Fatal(c.Services)
	}
	if _, err := os.Stat(filepath.Join(dir, "service-catalog.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyNormalizeAndAllows(t *testing.T) {
	cat := DefaultCatalog()
	p := AccessPolicy{
		AllowInternet: true,
		Services:      []string{"Lampac", "lampac", "internet", "git"},
		ServicesMode:  "list",
	}
	p.Normalize()
	if len(p.Services) != 2 {
		t.Fatalf("dedupe: %v", p.Services)
	}
	if err := p.Validate(cat); err != nil {
		t.Fatal(err)
	}
	if !p.Allows("lampac", cat) || !p.Allows("git", cat) {
		t.Fatal("should allow lampac/git")
	}
	if p.Allows("registry", cat) {
		t.Fatal("registry should be denied")
	}
	if !p.Allows("internet", cat) {
		t.Fatal("internet")
	}
	p.AllowInternet = false
	if p.Allows("internet", cat) {
		t.Fatal("internet denied")
	}
}

func TestPolicyModeAll(t *testing.T) {
	cat := DefaultCatalog()
	p := AccessPolicy{ServicesMode: "all", AllowInternet: true}
	ids := p.EffectiveServices(cat)
	if len(ids) < 3 {
		t.Fatal(ids)
	}
	for _, id := range ids {
		if id == "internet" {
			t.Fatal("egress should not be in effective internal list")
		}
	}
}

func TestPolicyUnknownService(t *testing.T) {
	cat := DefaultCatalog()
	p := AccessPolicy{Services: []string{"nope"}}
	p.Normalize()
	if err := p.Validate(cat); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpsertSoftDisable(t *testing.T) {
	c := DefaultCatalog()
	if err := c.Upsert(Service{ID: "foo", Title: "Foo", Kind: KindInternal}); err != nil {
		t.Fatal(err)
	}
	if err := c.SoftDisable("foo"); err != nil {
		t.Fatal(err)
	}
	s, ok := c.Get("foo")
	if !ok || !s.Disabled {
		t.Fatal(s)
	}
}

func TestInvalidServiceID(t *testing.T) {
	c := DefaultCatalog()
	if err := c.Upsert(Service{ID: "Bad ID", Kind: KindInternal}); err == nil {
		t.Fatal("expected invalid id")
	}
}
