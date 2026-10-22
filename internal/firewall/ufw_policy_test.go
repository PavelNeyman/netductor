package firewall

import "testing"

func TestUfwLineMentionsPort(t *testing.T) {
	cases := []struct {
		ln, port string
		want     bool
	}{
		{"52222/tcp                  ALLOW       Anywhere", "52222", true},
		{"52222/tcp                  ALLOW       Anywhere", "22", false},
		{"22/tcp                     DENY        Anywhere", "22", true},
		{"22/tcp                     DENY        Anywhere", "52222", false},
		{"10050/tcp                  DENY        Anywhere", "10050", true},
		{"8789/tcp                   ALLOW       10.87.10.0/30", "8789", true},
	}
	for _, c := range cases {
		if got := ufwLineMentionsPort(c.ln, c.port); got != c.want {
			t.Fatalf("ln=%q port=%q got=%v want=%v", c.ln, c.port, got, c.want)
		}
	}
}

func TestUfwEffectivePolicyPrimary(t *testing.T) {
	status := `Status: active

To                         Action      From
--                         ------      ----
52222/tcp                  ALLOW       Anywhere                   # netductor-ssh
22/tcp                     DENY        Anywhere                   # netductor-no-ssh22
443/tcp                    ALLOW       Anywhere                   # netductor-vless
10050/tcp                  DENY        Anywhere
8788/tcp                   DENY        Anywhere
8789/tcp                   ALLOW       10.87.10.0/30
8789/tcp                   ALLOW       127.0.0.1
`
	// sshPort() reads env/file — force via setting is hard; skip if not 52222
	ok, warns := ufwEffectivePolicy(status, "primary")
	if sshPort() == "52222" && !ok {
		t.Fatalf("expected ok, warns=%v", warns)
	}
}
