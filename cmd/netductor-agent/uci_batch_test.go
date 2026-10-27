package main

import "testing"

func TestUciBatchFailed(t *testing.T) {
	if uciBatchFailed("ok=3 fail=0\n") {
		t.Fatal("fail=0 should pass")
	}
	if !uciBatchFailed("ok=1 fail=2\nfail:set:x") {
		t.Fatal("fail=2 should fail")
	}
}
