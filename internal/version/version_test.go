package version

import "testing"

func TestReleaseNonEmpty(t *testing.T) {
	if Release == "" {
		t.Fatal("empty Release")
	}
}

func TestValidDotVersion(t *testing.T) {
	ok := []string{"0.9.493", "1.0.0", "10.2.3", "v0.9.493"}
	bad := []string{"", "1.2", "1.2.3.4", "01.2.3", "1.x.3"}
	for _, s := range ok {
		if !ValidDotVersion(s) {
			t.Fatalf("want valid %q", s)
		}
	}
	for _, s := range bad {
		if ValidDotVersion(s) {
			t.Fatalf("want invalid %q", s)
		}
	}
	// Less: strip v via verNorm only inside Less
	if !Less("0.9.490", "0.9.493") {
		t.Fatal("490 < 493")
	}
	if Less("0.9.493", "0.9.490") {
		t.Fatal("493 not < 490")
	}
}
