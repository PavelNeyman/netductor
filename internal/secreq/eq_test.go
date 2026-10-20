package secreq

import "testing"

func TestEq(t *testing.T) {
	if !Eq("abc", "abc") {
		t.Fatal("equal")
	}
	if Eq("abc", "abd") || Eq("", "x") || Eq("x", "") {
		t.Fatal("unequal or empty")
	}
}
