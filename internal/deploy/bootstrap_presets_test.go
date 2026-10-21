package deploy

import "testing"

func TestBootstrapPresetsUSB(t *testing.T) {
	ps := BootstrapPresets()
	if len(ps) < 1 || ps[0].ID != "usb-storage" {
		t.Fatalf("expected usb-storage preset")
	}
	if len(ps[0].Packages) < 3 {
		t.Fatalf("packages empty")
	}
	_, err := BootstrapPresetByID("nope")
	if err == nil {
		t.Fatal("expected error")
	}
	st := EvaluateBootstrapPresets(map[string]bool{}, false)
	if len(st) != 1 || !st[0].Selectable || st[0].Installed {
		t.Fatalf("expected selectable missing: %+v", st[0])
	}
	// mark all packages installed
	inst := map[string]bool{}
	for _, p := range ps[0].Packages {
		inst[p] = true
	}
	st = EvaluateBootstrapPresets(inst, true)
	if !st[0].Installed || st[0].Selectable {
		t.Fatalf("expected installed not selectable: %+v", st[0])
	}
}
