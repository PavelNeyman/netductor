package update

import "testing"

func TestMaskToken(t *testing.T) {
	if maskToken("short") != "****" {
		t.Fatal(maskToken("short"))
	}
	m := maskToken("ghp_abcdefghijklmnop")
	if m == "ghp_abcdefghijklmnop" || len(m) < 6 {
		t.Fatalf("%q", m)
	}
}

func TestGetTokenStatusNone(t *testing.T) {
	t.Setenv("NETDUCTOR_GITHUB_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	// cannot force TokenFile absent if host has one — only check struct shape
	st := GetTokenStatus()
	if st.Configured && st.Source == "" {
		t.Fatalf("%+v", st)
	}
	if !st.Configured && st.Source != "none" {
		t.Fatalf("%+v", st)
	}
}
