package addons

import "testing"

func TestStr(t *testing.T) {
	if str("a") != "a" || str(nil) != "" {
		t.Fatal()
	}
}
