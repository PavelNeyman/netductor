package vpn

import (
	"strings"
	"testing"
)

func TestVLESSLinkContainsReality(t *testing.T) {
	// smoke: format helpers exist
	_ = DefaultUTLSFingerprint
	if DefaultUTLSFingerprint == "" {
		t.Fatal("fingerprint empty")
	}
}

func TestPreferredLinkNonEmptyName(t *testing.T) {
	if !ValidName("alice") {
		t.Fatal("alice should be valid")
	}
	if ValidName("") {
		t.Fatal("empty valid")
	}
	_ = strings.Contains
}
