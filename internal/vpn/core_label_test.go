package vpn

import (
	"strings"
	"testing"
)

func TestSanitizeProfileFragment(t *testing.T) {
	if sanitizeProfileFragment("nd-control") != "nd-control" {
		t.Fatal("plain")
	}
	got := sanitizeProfileFragment("🇳🇱 nl-ams")
	if !strings.Contains(got, "nl-ams") {
		t.Fatalf("got %q", got)
	}
}

func TestCoreProfileLabelFallback(t *testing.T) {
	// Without CF-14 hostname, label must not be the old nd-primary singleton.
	l := coreProfileLabel()
	if l == "nd-primary" {
		t.Fatalf("legacy label %q", l)
	}
	if l == "" {
		t.Fatal("empty")
	}
}
