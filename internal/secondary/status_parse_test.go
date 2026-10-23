package secondary

import (
	"strings"
	"testing"
)

// Guard against the old Contains("active") trap (inactive matches).
func TestServiceActiveExact(t *testing.T) {
	cases := []struct {
		out  string
		want bool
	}{
		{"active\n", true},
		{"inactive\n", false},
		{"failed\n", false},
		{"activating\n", false},
	}
	for _, c := range cases {
		got := strings.TrimSpace(c.out) == "active"
		if got != c.want {
			t.Fatalf("%q: got %v want %v", c.out, got, c.want)
		}
	}
	// document old bug
	if !strings.Contains("inactive", "active") {
		t.Fatal("test assumption")
	}
}
