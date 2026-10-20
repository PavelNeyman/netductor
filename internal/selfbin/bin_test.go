package selfbin

import "testing"

func TestIsTestBinary(t *testing.T) {
	if !IsTestBinary("/tmp/incident.test") {
		t.Fatal("incident.test")
	}
	if IsTestBinary("/usr/local/bin/netductor") {
		t.Fatal("cli")
	}
	if CLI("/usr/local/bin/netductor") == "" {
		t.Fatal("empty")
	}
}
