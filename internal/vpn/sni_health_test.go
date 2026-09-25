package vpn

import "testing"

func TestSNIDialDown(t *testing.T) {
	if !SNIDialDown("dial: connection refused") {
		t.Fatal("expected down")
	}
	if SNIDialDown("sni=api.vk.me port_up dial_ms=1 (reality rejects plain TLS: i/o timeout)") {
		t.Fatal("port_up must not be down")
	}
}
