package vpn

import (
	"strings"
	"testing"
)

func TestSRProfilesContainFINAL(t *testing.T) {
	for _, id := range []string{SRProfileOperatorMobile, SRProfileOperatorFullProxy, SRProfileFamily} {
		body := BuildShadowrocketRoutingConfProfile(id)
		if !strings.Contains(body, "FINAL,PROXY") {
			t.Fatalf("%s: missing FINAL,PROXY", id)
		}
		if !strings.Contains(body, "[Rule]") {
			t.Fatalf("%s: missing [Rule]", id)
		}
		if !strings.Contains(body, "[General]") {
			t.Fatalf("%s: missing [General]", id)
		}
	}
}

func TestOperatorMobileHasRUAndFamilyHasRU(t *testing.T) {
	m := BuildShadowrocketRoutingConfProfile(SRProfileOperatorMobile)
	if !strings.Contains(m, "GEOIP,RU,DIRECT") {
		t.Fatal("mobile: expected GEOIP RU DIRECT")
	}
	if !strings.Contains(m, "10.9.8.0/24") {
		t.Fatal("mobile: expected home LAN")
	}
	if !strings.Contains(m, "10.88.0.0/24,PROXY") && !strings.Contains(m, "IP-CIDR,10.88.0.0/24,PROXY") {
		t.Fatal("mobile: expected service-net PROXY")
	}
	f := BuildShadowrocketRoutingConfProfile(SRProfileFamily)
	if !strings.Contains(f, "GEOIP,RU,DIRECT") {
		t.Fatal("family: expected GEOIP RU")
	}
	if strings.Contains(f, "10.88.0.0/24") {
		t.Fatal("family: must not include service-net")
	}
	full := BuildShadowrocketRoutingConfProfile(SRProfileOperatorFullProxy)
	if strings.Contains(full, "GEOIP,RU,DIRECT") {
		t.Fatal("fullproxy: must NOT have client RU DIRECT")
	}
	if !strings.Contains(full, "10.9.8.0/24") {
		t.Fatal("fullproxy: home LAN DIRECT expected")
	}
}

func TestNormalizeSRProfile(t *testing.T) {
	if NormalizeSRProfile("full") != SRProfileOperatorFullProxy {
		t.Fatal("full alias")
	}
	if NormalizeSRProfile("family") != SRProfileFamily {
		t.Fatal("family")
	}
}
