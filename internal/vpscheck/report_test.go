package vpscheck

import "testing"

func TestParseProfile(t *testing.T) {
	if ParseProfile("secondary") != ProfileSecondary {
		t.Fatal("secondary")
	}
	if ParseProfile("ru") != ProfileSecondary {
		t.Fatal("ru")
	}
	if ParseProfile("primary") != ProfilePrimary {
		t.Fatal("primary")
	}
}
