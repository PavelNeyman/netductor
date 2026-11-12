package vpn

import (
	"os"
	"testing"
)

func TestUplinkMultiplexObjectCanonical(t *testing.T) {
	t.Setenv("NETDUCTOR_UPLINK_MUX", "on")
	m := uplinkMultiplexObject()
	if m == nil {
		t.Fatal("expected mux object")
	}
	if m["enabled"] != true {
		t.Fatalf("enabled: %v", m["enabled"])
	}
	if _, ok := m["max_streams"]; ok {
		t.Fatal("max_streams must not be set (conflicts with max_connections)")
	}
	if m["max_connections"] != 8 {
		t.Fatalf("max_connections: %v", m["max_connections"])
	}
	if m["min_streams"] != 4 {
		t.Fatalf("min_streams: %v", m["min_streams"])
	}
	if m["protocol"] != "h2mux" {
		t.Fatalf("protocol: %v", m["protocol"])
	}
	if m["padding"] != true {
		t.Fatalf("padding: %v want true (primary inbound requires it)", m["padding"])
	}
}

func TestUplinkMultiplexObjectOff(t *testing.T) {
	t.Setenv("NETDUCTOR_UPLINK_MUX", "off")
	if uplinkMultiplexObject() != nil {
		t.Fatal("expected nil when off")
	}
}

func TestUplinkMultiplexObjectSmux(t *testing.T) {
	t.Setenv("NETDUCTOR_UPLINK_MUX", "smux")
	m := uplinkMultiplexObject()
	if m == nil || m["protocol"] != "smux" {
		t.Fatalf("got %#v", m)
	}
	if _, ok := m["max_streams"]; ok {
		t.Fatal("max_streams must not be set")
	}
	_ = os.Unsetenv // keep import used on older go if needed
}
