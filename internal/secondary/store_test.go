package secondary

import "testing"

func TestFindByTokenRejectEmpty(t *testing.T) {
	if FindByToken("") != nil {
		t.Fatal("empty token must not match")
	}
	if FindByToken("short") != nil {
		t.Fatal("short token must not match")
	}
}

func TestHeartbeatDoesNotEraseLastKnownPublicIP(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))

	_, token, err := IssueToken("secondary-test")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	first, _, err := Heartbeat(token, HeartbeatIn{PublicIP: "195.140.147.20"})
	if err != nil {
		t.Fatalf("initial heartbeat: %v", err)
	}
	if first.PublicIP != "195.140.147.20" {
		t.Fatalf("initial public IP = %q", first.PublicIP)
	}

	afterProbeFailure, _, err := Heartbeat(token, HeartbeatIn{PublicIP: ""})
	if err != nil {
		t.Fatalf("heartbeat after probe failure: %v", err)
	}
	if afterProbeFailure.PublicIP != "195.140.147.20" {
		t.Fatalf("empty heartbeat erased last known public IP: got %q", afterProbeFailure.PublicIP)
	}

	afterAddressChange, _, err := Heartbeat(token, HeartbeatIn{PublicIP: "195.140.147.21"})
	if err != nil {
		t.Fatalf("heartbeat after address change: %v", err)
	}
	if afterAddressChange.PublicIP != "195.140.147.21" {
		t.Fatalf("non-empty public IP update ignored: got %q", afterAddressChange.PublicIP)
	}
}
