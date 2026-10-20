package edgeagent

import "testing"

func TestSplitArg(t *testing.T) {
	u, s, r := SplitArg("https://x/a|abc|confirm=yes")
	if u != "https://x/a" || s != "abc" || r != "confirm=yes" {
		t.Fatalf("%q %q %q", u, s, r)
	}
	u, s, r = SplitArg("https://only")
	if u != "https://only" || s != "" || r != "" {
		t.Fatal(u, s, r)
	}
}

func TestSysupgradeAllowed(t *testing.T) {
	if SysupgradeAllowed("url|sha") {
		t.Fatal()
	}
	if !SysupgradeAllowed("url|sha|confirm=yes") {
		t.Fatal()
	}
}
