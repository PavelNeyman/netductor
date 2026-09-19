package vpn

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSingBoxClientJSON_RUDirect(t *testing.T) {
	e := ClientEndpoints{
		Name: "t", UUID: "00000000-0000-0000-0000-000000000001",
		CoreHost: "1.2.3.4", CorePort: 443, CoreSNI: "api.vk.me",
		CorePBK: "testpbk", CoreSID: "abcd",
	}
	b, err := SingBoxClientJSON(e)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, need := range []string{"gosuslugi.ru", "geoip-ru", "77.88.8.8", `"outbound": "direct"`} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in config", need)
		}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
}

func TestShadowrocketRoutingHasGEOIP(t *testing.T) {
	rules := ShadowrocketRuDirectRules()
	joined := strings.Join(rules, "\n")
	if !strings.Contains(joined, "GEOIP,RU,DIRECT") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "DOMAIN-KEYWORD,gosuslugi,DIRECT") {
		t.Fatal("keyword")
	}
}
