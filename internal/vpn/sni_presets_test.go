package vpn

import "testing"

func TestListSNIPresets(t *testing.T) {
	p := ListSNIPresets()
	if len(p) < 2 {
		t.Fatal("expected presets")
	}
}
