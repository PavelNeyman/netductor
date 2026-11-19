package main

import "testing"

func TestSsAddrHasPort(t *testing.T) {
	if !ssAddrHasPort("127.0.0.1:53", "53") {
		t.Fatal("53")
	}
	if ssAddrHasPort("0.0.0.0:53623", "53") {
		t.Fatal("must not match 53623 as 53")
	}
	if !ssAddrHasPort("0.0.0.0:53", "53") {
		t.Fatal("all 53")
	}
	if ssAddrHasPort("127.0.0.1:853", "53") {
		t.Fatal("853")
	}
}
