package edge

import (
	"testing"
	"time"
)

func TestDeriveSiblingCIDRs(t *testing.T) {
	g, i := deriveSiblingCIDRs("10.120.0.0/24")
	if g != "10.120.1.0/24" || i != "10.120.2.0/24" {
		t.Fatalf("got %s %s", g, i)
	}
}

func TestBuildSiteSummaryZones(t *testing.T) {
	d := Device{
		DeviceID: "shop-1",
		Hostname: "TvoiParfum",
		Status:   StatusApproved,
		WANIP:    "203.0.113.45",
		LastSeen: time.Now().Unix(),
		Overlay: map[string]any{
			"lan_ip": "10.120.0.1",
			"guest":  map[string]any{"enabled": true, "ssid": "Guest"},
			"iot":    map[string]any{"enabled": true, "ssid": "IoT"},
		},
		GuestActive: 7,
		DeviceCount: 24,
		UptimeSec:   7*86400 + 14*3600 + 22*60,
		LatencyMs:   12,
	}
	s := BuildSiteSummary(d)
	if s.Name != "TvoiParfum" {
		t.Fatal(s.Name)
	}
	if len(s.Zones) != 3 {
		t.Fatalf("zones %d", len(s.Zones))
	}
	if s.Zones[0].CIDR != "10.120.0.0/24" {
		t.Fatal(s.Zones[0].CIDR)
	}
	if !s.Zones[1].Enabled || s.Zones[2].CIDR != "10.120.2.0/24" {
		t.Fatalf("%+v", s.Zones)
	}
}
