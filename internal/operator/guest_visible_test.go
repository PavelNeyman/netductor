package operator

import "testing"

func TestGuestSSIDVisible(t *testing.T) {
	cases := map[string]bool{
		"":        true,
		"0":       true,
		"false":   true,
		"no":      true,
		"visible": true,
		"1":       false,
		"true":    false,
		"yes":     false,
		"hidden":  false,
		"hide":    false,
		"YES":     false,
		" 0 ":     true,
	}
	for in, want := range cases {
		if got := guestSSIDVisible(in); got != want {
			t.Fatalf("guestSSIDVisible(%q)=%v want %v", in, got, want)
		}
	}
}
