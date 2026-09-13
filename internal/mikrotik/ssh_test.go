package mikrotik

import "testing"

func TestPushRSCRequiresHost(t *testing.T) {
	err := PushRSC("", "admin", "x", nil, "/system identity print", 22)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientRSCContainsIdentity(t *testing.T) {
	s := ClientRSC("mt-home", "1.2.3.4", "", "note")
	if s == "" || len(s) < 10 {
		t.Fatal(s)
	}
}
