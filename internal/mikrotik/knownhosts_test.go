package mikrotik

import "testing"

func TestKnownHostsPath(t *testing.T) {
	p := knownHostsPath()
	if p == "" {
		t.Fatal("empty path")
	}
}
