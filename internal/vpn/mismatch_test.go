package vpn

import "testing"

func TestParseMismatchCount(t *testing.T) {
	log := "ok line\ninbound flow mismatch from 1.2.3.4:1\nother\nflow mismatch from 1.2.3.4:2"
	log = "ok line" + string([]byte{10}) + "inbound flow mismatch from 1.2.3.4:1" + string([]byte{10}) + "flow mismatch from 5.6.7.8:9"
	if ParseMismatchCount(log) != 2 {
		t.Fatal(ParseMismatchCount(log))
	}
}
