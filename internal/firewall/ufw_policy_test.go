package firewall

import "testing"

func TestParseUfwNumberedOrder(t *testing.T) {
	status := `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 52222/tcp                  ALLOW IN    Anywhere
[ 2] 22/tcp                     ALLOW IN    Anywhere
[ 3] 22/tcp                     DENY IN     Anywhere
[ 4] 443/tcp                    ALLOW IN    Anywhere
[ 5] 10050/tcp                  DENY IN     Anywhere
[ 6] 8788/tcp                   DENY IN     Anywhere
[ 7] 8789/tcp                   ALLOW IN    10.87.10.0/30
[ 8] 8789/tcp                   ALLOW IN    127.0.0.1
[ 9] 22/tcp (v6)                DENY IN     Anywhere (v6)
`
	rules := parseUfwNumbered(status)
	if len(rules) < 7 {
		t.Fatalf("rules=%d", len(rules))
	}
	// v6 skipped
	for _, r := range rules {
		if r.Proto == "udp" {
			t.Fatal("udp")
		}
	}
	act, found := firstTCPActionFromWAN(rules, "22")
	if !found || act != "ALLOW" {
		t.Fatalf("first WAN 22 should be ALLOW (stale), got found=%v act=%s", found, act)
	}
	ok, warns := ufwEffectivePolicy(status, "primary")
	if ok {
		t.Fatalf("expected fail on early ALLOW 22, warns=%v", warns)
	}
}

func TestUfwNumberedGoodPrimary(t *testing.T) {
	// sshPort() defaults often 52222
	status := `Status: active
[ 1] 52222/tcp                  ALLOW IN    Anywhere
[ 2] 22/tcp                     DENY IN     Anywhere
[ 3] 443/tcp                    ALLOW IN    Anywhere
[ 4] 10050/tcp                  DENY IN     Anywhere
[ 5] 8788/tcp                   DENY IN     Anywhere
[ 6] 8789/tcp                   ALLOW IN    10.87.10.0/30
[ 7] 8789/tcp                   ALLOW IN    10.87.11.0/30
[ 8] 8789/tcp                   ALLOW IN    127.0.0.1
`
	if sshPort() != "52222" {
		t.Skip("ssh port not 52222 in test env")
	}
	ok, warns := ufwEffectivePolicy(status, "primary")
	if !ok {
		t.Fatalf("want ok, warns=%v", warns)
	}
}

func TestUfwLineMentionsPort(t *testing.T) {
	if !ufwLineMentionsPort("52222/tcp ALLOW Anywhere", "52222") {
		t.Fatal("52222")
	}
	if ufwLineMentionsPort("52222/tcp ALLOW Anywhere", "22") {
		t.Fatal("must not match 22 inside 52222")
	}
}
