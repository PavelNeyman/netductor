package operator

import "testing"

func TestSecondarySpecToDeployPortAndPeer(t *testing.T) {
	s := SecondarySpec{
		PrimaryHost: "198.51.100.1", PrimarySSHPort: 2200, PeerID: "peer-b2",
		SecondaryHost: "203.0.113.9",
	}
	o := s.toDeploy()
	if o.PrimarySSHPort != 2200 || o.PeerID != "peer-b2" {
		t.Fatalf("toDeploy dropped port/peer: %+v", o)
	}
}

func TestSecondaryFromFieldsPortAndPeer(t *testing.T) {
	s := SecondaryFromFields(func(k string) string {
		switch k {
		case "host":
			return "203.0.113.9"
		case "primary_ssh_port":
			return "2200"
		case "peer_id":
			return "peer-b2"
		case "secondary_ssh_key":
			return "/k"
		}
		return ""
	})
	if s.PrimarySSHPort != 2200 || s.PeerID != "peer-b2" {
		t.Fatalf("fields: %+v", s)
	}
}
