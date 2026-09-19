package vpn

import "testing"

func TestRuDirectHasGosuslugi(t *testing.T) {
	found := false
	for _, s := range RuDirectSuffixes() {
		if s == "gosuslugi.ru" || s == "ru" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected gosuslugi/ru suffixes")
	}
	kw := false
	for _, k := range RuDirectKeywords() {
		if k == "gosuslugi" {
			kw = true
		}
	}
	if !kw {
		t.Fatal("keyword gosuslugi")
	}
	rules := ShadowrocketRuDirectRules()
	if len(rules) < 10 {
		t.Fatalf("too few rules %d", len(rules))
	}
}
