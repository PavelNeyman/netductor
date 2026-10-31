package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUfwRuleToken(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"52222/tcp                  ALLOW       Anywhere                   # netductor-ssh", "52222/tcp ALLOW Anywhere"},
		{"22/tcp                     DENY        Anywhere                   # netductor-no-ssh22", "22/tcp DENY Anywhere"},
		{"8789/tcp                   ALLOW       10.87.10.0/30              # netductor-api", "8789/tcp ALLOW 10.87.10.0/30"},
		{"Status: active", ""},
		{"To                         Action      From", ""},
		{"--                         ------      ----", ""},
	}
	for _, c := range cases {
		if got := ufwRuleToken(c.in); got != c.want {
			t.Fatalf("in=%q got=%q want=%q", c.in, got, c.want)
		}
	}
}

func TestRuleSetDiffUfw(t *testing.T) {
	dir := t.TempDir()
	snap := filepath.Join(dir, "ufw.txt")
	base := `Status: active
To                         Action      From
--                         ------      ----
52222/tcp                  ALLOW       Anywhere
22/tcp                     DENY        Anywhere
443/tcp                    ALLOW       Anywhere
`
	live := base + "9999/tcp                    ALLOW       Anywhere\n"
	// missing 443 vs extra 9999
	liveMissing443 := `Status: active
52222/tcp                  ALLOW       Anywhere
22/tcp                     DENY        Anywhere
9999/tcp                    ALLOW       Anywhere
`
	if err := os.WriteFile(snap, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	extra, missing := ruleSetDiff(snap, liveMissing443, ufwRuleToken)
	if len(extra) != 1 || extra[0] != "9999/tcp ALLOW Anywhere" {
		t.Fatalf("extra=%v", extra)
	}
	if len(missing) != 1 || missing[0] != "443/tcp ALLOW Anywhere" {
		t.Fatalf("missing=%v", missing)
	}
	extra2, missing2 := ruleSetDiff(snap, live, ufwRuleToken)
	if len(extra2) != 1 || len(missing2) != 0 {
		t.Fatalf("extra2=%v missing2=%v", extra2, missing2)
	}
}

func TestIptablesChainPolicyToken(t *testing.T) {
	if got := iptablesLineToken(":INPUT DROP [0:0]"); got != ":INPUT DROP" {
		t.Fatalf("got %q", got)
	}
	if got := iptablesLineToken(":INPUT ACCEPT [12:34]"); got != ":INPUT ACCEPT" {
		t.Fatalf("got %q", got)
	}
	a, b := ruleSetDiff("/dev/null", ":INPUT DROP [0:0]\n-A INPUT -j ACCEPT\n", iptablesLineToken)
	_ = a
	_ = b
}

func TestIptablesPolicyDriftDetected(t *testing.T) {
	dir := t.TempDir()
	bp := filepath.Join(dir, "iptables.txt")
	base := ":INPUT DROP [0:0]\n-A INPUT -j ACCEPT\n"
	live := ":INPUT ACCEPT [9:9]\n-A INPUT -j ACCEPT\n"
	if err := os.WriteFile(bp, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	added, removed := ruleSetDiff(bp, live, iptablesLineToken)
	ok := false
	for _, r := range removed {
		if r == ":INPUT DROP" {
			ok = true
		}
	}
	for _, a := range added {
		if a == ":INPUT ACCEPT" {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("expected INPUT policy drift; added=%v removed=%v", added, removed)
	}
}
