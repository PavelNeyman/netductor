package main

import "testing"

func TestSecondaryLinkComplete(t *testing.T) {
	if secondaryLinkComplete("203.0.113.9", "pbk", "") {
		t.Fatal("empty SID must not link")
	}
	if !secondaryLinkComplete("203.0.113.9", "pbk", "sid") {
		t.Fatal("complete should link")
	}
	if secondaryLinkComplete("", "pbk", "sid") || secondaryLinkComplete("203.0.113.9", "", "sid") {
		t.Fatal("empty ip/pbk must not link")
	}
}
