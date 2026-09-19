package update

import "testing"

func TestNewer(t *testing.T) {
	if !Newer("v0.8.0", "0.7.39-dev") {
		t.Fatal("0.8 > 0.7")
	}
	if Newer("0.8.0", "0.8.0") {
		t.Fatal("equal")
	}
	if Newer("0.7.9", "0.8.0") {
		t.Fatal("older")
	}
}
