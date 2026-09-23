package edgeagent

import (
	"strings"
	"testing"
)

func TestVLESSClientConfigSocks(t *testing.T) {
	link := "vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.cloudflare.com&fp=chrome&pbk=PUB&sid=abcd&type=tcp#t"
	b, err := VLESSClientConfig(link, "socks")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "mixed") || !strings.Contains(s, "7890") {
		t.Fatal(s)
	}
}

func TestVLESSClientConfigTun(t *testing.T) {
	link := "vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.cloudflare.com&fp=chrome&pbk=PUB&sid=abcd&type=tcp#t"
	b, err := VLESSClientConfig(link, "tun")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "tun") {
		t.Fatal(string(b))
	}
}

func TestVLESSClientConfigDefaultTun(t *testing.T) {
	link := "vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.example.com&fp=chrome&pbk=abc&sid=abcd#t"
	b, err := VLESSClientConfig(link, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"type": "tun"`) && !strings.Contains(string(b), `"type":"tun"`) {
		t.Fatalf("expected tun default, got %s", b)
	}
}
