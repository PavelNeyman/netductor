package dnsblock

import "testing"

func TestDefaultBlockyDenyURLs(t *testing.T) {
	u := DefaultBlockyDenyURLs()
	if len(u) < 1 {
		t.Fatal("empty")
	}
	for _, x := range u {
		if len(x) < 8 || x[:4] != "http" {
			t.Fatalf("%q", x)
		}
	}
}

func TestCatalogNonEmpty(t *testing.T) {
	if len(Catalog()) < 1 {
		t.Fatal("catalog")
	}
}
