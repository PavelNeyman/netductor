package main

import (
	"encoding/base64"
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
