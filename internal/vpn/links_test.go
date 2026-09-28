package vpn

import "testing"

func TestValidName(t *testing.T) {
	if !ValidName("operator") {
		t.Fatal()
	}
	if ValidName("bad name") {
		t.Fatal("space")
	}
}
