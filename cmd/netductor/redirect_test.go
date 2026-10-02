package main

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestAllowedDeepLink(t *testing.T) {
	ok := []string{
		"shadowrocket://add/vless://x",
		"happ://add/vless://x",
		"incy://add/vless://x",
		"vless://uuid@host:443?security=reality",
		"hysteria2://pw@host:8443?sni=x",
	}
	for _, s := range ok {
		if !allowedDeepLink(s) {
			t.Fatalf("expected allowed: %s", s)
		}
	}
	bad := []string{"https://evil.com", "javascript:alert(1)", "file:///etc/passwd", ""}
	for _, s := range bad {
		if allowedDeepLink(s) {
			t.Fatalf("expected denied: %s", s)
		}
	}
}

func TestRedirectRoundTripEncoding(t *testing.T) {
	deep := "shadowrocket://add/vless://b74ef933-d1f2-4231-8b67-d701dc28cd6b@vpn.example:443?security=reality&pbk=x"
	enc := base64.RawURLEncoding.EncodeToString([]byte(deep))
	b, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil || string(b) != deep {
		t.Fatalf("roundtrip: %v %q", err, b)
	}
	if !allowedDeepLink(string(b)) {
		t.Fatal("decoded not allowed")
	}
}

func TestAllowSubRequestRateLimit(t *testing.T) {
	// isolate map for this IP
	ip := "203.0.113.50"
	subRLMu.Lock()
	delete(subRL, ip)
	subRLMu.Unlock()
	for i := 0; i < 60; i++ {
		if !allowSubRequest(ip) {
			t.Fatalf("allowed until 60, failed at %d", i+1)
		}
	}
	if allowSubRequest(ip) {
		t.Fatal("61st must be denied")
	}
	// other IP still ok
	ip2 := "203.0.113.51"
	subRLMu.Lock()
	delete(subRL, ip2)
	subRLMu.Unlock()
	if !allowSubRequest(ip2) {
		t.Fatal("other IP should pass")
	}
}

func TestSubClientIP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/sub/x", nil)
	r.RemoteAddr = "198.51.100.9:12345"
	if got := subClientIP(r); got != "198.51.100.9" {
		t.Fatalf("got %q", got)
	}
}
