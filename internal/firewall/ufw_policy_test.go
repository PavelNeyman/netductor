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
[ 2] 52222/tcp (v6)             ALLOW IN    Anywhere (v6)
[ 3] 22/tcp                     DENY IN     Anywhere
[ 4] 22/tcp (v6)                DENY IN     Anywhere (v6)
[ 5] 443/tcp                    ALLOW IN    Anywhere
[ 6] 443/tcp (v6)               ALLOW IN    Anywhere (v6)
[ 7] 10050/tcp                  DENY IN     Anywhere
[ 8] 10050/tcp (v6)             DENY IN     Anywhere (v6)
[ 9] 10051/tcp                  DENY IN     Anywhere
[10] 10051/tcp (v6)             DENY IN     Anywhere (v6)
[11] 8443/tcp                   ALLOW IN    Anywhere
[12] 8443/tcp (v6)              ALLOW IN    Anywhere (v6)
[13] 80/tcp                     ALLOW IN    Anywhere
[14] 80/tcp (v6)                ALLOW IN    Anywhere (v6)
[15] 8787/tcp                   DENY IN     Anywhere
[16] 8787/tcp (v6)              DENY IN     Anywhere (v6)
[17] 8445/tcp                   DENY IN     Anywhere
[18] 8445/tcp (v6)              DENY IN     Anywhere (v6)
[19] 8788/tcp                   DENY IN     Anywhere
[20] 8788/tcp (v6)              DENY IN     Anywhere (v6)
[21] 9443/tcp                   ALLOW IN    10.87.10.0/30
[22] 9443/tcp                   ALLOW IN    10.87.11.0/30
[23] 9443/tcp                   DENY IN     Anywhere
[24] 9443/tcp (v6)              DENY IN     Anywhere (v6)
[25] 8789/tcp                   ALLOW IN    10.87.10.0/30
[26] 8789/tcp                   ALLOW IN    10.87.11.0/30
[27] 9118/tcp                   DENY IN     Anywhere
[28] 9118/tcp (v6)              DENY IN     Anywhere (v6)
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

func TestHasExactWSSPeerAllowRequiresHostScopedRule(t *testing.T) {
	peer := wssExpectation{Port: "8444", Source: "92.255.77.253"}
	global := parseUfwNumbered("Status: active\n[ 1] 8444/tcp ALLOW IN Anywhere\n")
	if hasExactWSSPeerAllow(global, peer) {
		t.Fatal("global WSS allow must not count as a peer-scoped rule")
	}
	exact := parseUfwNumbered("Status: active\n[ 1] 8444/tcp ALLOW IN 92.255.77.253\n")
	if !hasExactWSSPeerAllow(exact, peer) {
		t.Fatal("exact peer address should count")
	}
	hostCIDR := parseUfwNumbered("Status: active\n[ 1] 8444/tcp ALLOW IN 92.255.77.253/32\n")
	if !hasExactWSSPeerAllow(hostCIDR, peer) {
		t.Fatal("exact /32 peer address should count")
	}
	broad := parseUfwNumbered("Status: active\n[ 1] 8444/tcp ALLOW IN 92.255.77.0/24\n")
	if hasExactWSSPeerAllow(broad, peer) {
		t.Fatal("broader subnet must not count as an exact peer rule")
	}
}

func TestSecondaryRestrictedAllowPreflight(t *testing.T) {
	for _, status := range []string{
		"Status: active\n[ 1] 80/tcp ALLOW IN Anywhere\n",
		"Status: inactive\n[ 1] 8789/tcp ALLOW IN 10.87.10.2\n",
		"Status: active\n[ 1] 8787/tcp ALLOW IN 192.0.2.10\n",
	} {
		if err := preflightSecondaryRestrictedAllows(status); err == nil {
			t.Fatalf("expected pre-existing restricted-port ALLOW to fail: %q", status)
		}
	}
	clean := "Status: active\n[ 1] 80/tcp DENY IN Anywhere\n[ 2] 8789/tcp DENY IN Anywhere\n"
	if err := preflightSecondaryRestrictedAllows(clean); err != nil {
		t.Fatalf("deny-only rules should pass preflight: %v", err)
	}
	incomplete := "Status: active\n[ 1] 80/tcp ALLOW IN Anywhere\nnot-numbered"
	if err := preflightSecondaryRestrictedAllows(incomplete); err == nil {
		t.Fatal("incompletely parsed numbered rules must fail closed")
	}
}

func TestSecondaryInactiveSavedAllowPreflight(t *testing.T) {
	for _, rule := range []string{
		"ufw allow 80/tcp",
		"ufw allow from 192.0.2.10 to any port 8789 proto tcp",
		"ufw allow 8787",
	} {
		if err := preflightSecondaryAddedRules("Added user rules:\\n"+rule+"\\n"); err == nil {
			t.Fatalf("expected saved restricted allow to fail: %q", rule)
		}
	}
	for _, rule := range []string{"ufw deny 80/tcp", "ufw allow 443/tcp", "ufw allow from 10.87.10.2 to any port 8445 proto tcp"} {
		if err := preflightSecondaryAddedRules(rule); err != nil {
			t.Fatalf("unrelated/non-allow rule should pass: %q: %v", rule, err)
		}
	}
}
