package deploy

import "testing"

func TestValidUSBPkgNameTildeVersion(t *testing.T) {
	ok := "block-mount-2026.05.23~16718b6e-r1.apk"
	if !validUSBPkgName(ok) {
		t.Fatalf("expected valid: %s", ok)
	}
	if validUSBPkgName("../evil.apk") {
		t.Fatal("path rejected")
	}
	if validUSBPkgName("x*.apk") {
		t.Fatal("glob rejected")
	}
}
