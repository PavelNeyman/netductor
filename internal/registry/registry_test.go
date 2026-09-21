package registry

import "testing"

func TestAddrDefault(t *testing.T) {
	if Addr() == "" {
		t.Fatal("empty addr")
	}
	h, p := HostPort()
	if p == "" || h == "" {
		t.Fatal(h, p)
	}
}
