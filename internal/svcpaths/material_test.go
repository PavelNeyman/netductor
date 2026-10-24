package svcpaths

import "testing"

func TestForSecondaryStripsPrimaryPrivates(t *testing.T) {
	m := &Material{
		SPPrivP: "a", SPPubP: "b", SPPrivS: "c", SPPubS: "d",
		PSPrivP: "e", PSPubP: "f", PSPrivS: "g", PSPubS: "h",
		Primary: "1.1.1.1", Secondary: "2.2.2.2",
	}
	s := m.ForSecondary()
	if s.SPPrivP != "" || s.PSPrivP != "" {
		t.Fatal("primary privates leaked")
	}
	if s.SPPrivS != "c" || s.PSPrivS != "g" {
		t.Fatal("secondary privates missing")
	}
}
