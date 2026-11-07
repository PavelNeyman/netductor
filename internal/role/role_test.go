package role

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"primary":   Control,
		"PRIMARY":   Control,
		"control":   Control,
		"secondary": Egress,
		"egress":    Egress,
		"relay":     Egress,
		"edge":      Edge,
		"build":     Build,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", in, got, want)
		}
	}
}

func TestAcceptsRoleFile(t *testing.T) {
	if !AcceptsRoleFile("primary") || !AcceptsRoleFile("control") {
		t.Fatal("control aliases")
	}
	if !AcceptsRoleFile("secondary") || !AcceptsRoleFile("egress") {
		t.Fatal("egress aliases")
	}
	if AcceptsRoleFile("edge") {
		t.Fatal("edge not yet a host role file value")
	}
}

func TestLegacyWrite(t *testing.T) {
	if LegacyWrite("control") != Primary {
		t.Fatal(LegacyWrite("control"))
	}
	if LegacyWrite("egress") != Secondary {
		t.Fatal(LegacyWrite("egress"))
	}
}
