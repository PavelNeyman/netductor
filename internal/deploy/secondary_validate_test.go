package deploy

import (
	"encoding/json"
	"testing"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func TestDeploySecondaryRequiresIPv4Host(t *testing.T) {
	err := DeploySecondary(SecondaryOpts{
		PrimaryHost:   "1.2.3.4",
		PrimaryKey:    "/nonexistent",
		SecondaryHost: "router.local",
		SecondaryPass: "x",
	})
	if err == nil || err.Error() == "" {
		t.Fatal("expected error for hostname")
	}
	if want := "public IP"; !contains(err.Error(), want) && !contains(err.Error(), "must be a public") {
		// soft check — message should mention IP requirement
		t.Log(err)
	}

	err = DeploySecondary(SecondaryOpts{
		PrimaryHost:   "1.2.3.4",
		PrimaryKey:    "/nonexistent",
		SecondaryHost: "2001:db8::1",
		SecondaryPass: "x",
	})
	if err == nil {
		t.Fatal("expected error for IPv6")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}

func TestSecondaryPackNameIsScopedToHost(t *testing.T) {
	first := secondaryPackName("195.140.147.20")
	second := secondaryPackName("92.255.77.253")
	if first != "secondary-195-140-147-20" {
		t.Fatalf("unexpected first pack name %q", first)
	}
	if second != "secondary-92-255-77-253" {
		t.Fatalf("unexpected second pack name %q", second)
	}
	if first == second {
		t.Fatal("different Secondary hosts must not share a pending pack name")
	}
}

func peerRegistryJSON(t *testing.T, peers ...svcpaths.ServicePeer) string {
	t.Helper()
	b, err := json.Marshal(svcpaths.PeersIndex{Version: 1, Peers: peers})
	if err != nil { t.Fatal(err) }
	return string(b)
}

func TestSecondaryPeerPreflight(t *testing.T) {
	slots := svcpaths.SlotTable()
	peerID, host := "peer-195-140-147-20", "195.140.147.20"
	if err := validateSecondaryPeerPreflight(peerRegistryJSON(t), peerID, host); err != nil {
		t.Fatalf("empty registry should have capacity: %v", err)
	}
	existing := svcpaths.ServicePeer{ID: peerID, PublicIP: host, State: "prepared", Slot: slots[1]}
	if err := validateSecondaryPeerPreflight(peerRegistryJSON(t, existing), peerID, host); err != nil {
		t.Fatalf("matching existing peer should be idempotent: %v", err)
	}
	existing.PublicIP = "195.140.147.21"
	if err := validateSecondaryPeerPreflight(peerRegistryJSON(t, existing), peerID, host); err == nil {
		t.Fatal("peer ID bound to another IP must fail")
	}
	existing.ID, existing.PublicIP = "other-peer", host
	if err := validateSecondaryPeerPreflight(peerRegistryJSON(t, existing), peerID, host); err == nil {
		t.Fatal("public IP bound to another peer must fail")
	}
	existing.ID, existing.PublicIP, existing.Slot = peerID, host, svcpaths.PeerSlot{Index: 1}
	if err := validateSecondaryPeerPreflight(peerRegistryJSON(t, existing), peerID, host); err == nil {
		t.Fatal("matching peer with inconsistent slot tuple must fail")
	}
	full := make([]svcpaths.ServicePeer, 0, len(slots))
	for i, slot := range slots {
		full = append(full, svcpaths.ServicePeer{ID: "occupied-"+string(rune('a'+i)), PublicIP: "198.51.100."+string(rune('1'+i)), State: "prepared", Slot: slot})
	}
	if err := validateSecondaryPeerPreflight(peerRegistryJSON(t, full...), peerID, host); err == nil {
		t.Fatal("full registry must fail before provision")
	}
}
