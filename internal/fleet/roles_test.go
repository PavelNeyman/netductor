package fleet

import "testing"

func TestEmptyHelper(t *testing.T) {
	if empty("", "x") != "x" || empty("a", "x") != "a" {
		t.Fatal()
	}
}

func TestDefaultPolicy(t *testing.T) { _ = defaultPolicy() }
