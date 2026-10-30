package main

import "testing"

func TestParseSSHTrustArgs(t *testing.T) {
	a, err := parseSSHTrustArgs([]string{"1.2.3.4", "--port", "52222", "--show"})
	if err != nil || !a.showOnly || a.host != "1.2.3.4" || a.port != "52222" {
		t.Fatalf("%+v %v", a, err)
	}
	a, err = parseSSHTrustArgs([]string{"1.2.3.4", "--fingerprint", "SHA256:abc"})
	if err != nil || a.fingerprint != "SHA256:abc" || a.showOnly {
		t.Fatalf("%+v %v", a, err)
	}
	if _, err := parseSSHTrustArgs([]string{"1.2.3.4"}); err == nil {
		t.Fatal("want fingerprint required")
	}
	if _, err := parseSSHTrustArgs([]string{"1.2.3.4", "--show", "--fingerprint", "SHA256:x"}); err == nil {
		t.Fatal("want refuse both flags")
	}
	if _, err := parseSSHTrustArgs([]string{"--show"}); err == nil {
		t.Fatal("want host required")
	}
	if _, err := parseSSHTrustArgs([]string{"h", "--port"}); err == nil {
		t.Fatal("want port value")
	}
	if _, err := parseSSHTrustArgs([]string{"h", "--unknown"}); err == nil {
		t.Fatal("want unknown flag")
	}
}
