package policy

import "testing"

func TestApplyPreset(t *testing.T) {
	p := ApplyPreset(PresetFull, DefaultUserPolicy())
	if p.ServicesMode != "all" || !p.AllowInternet {
		t.Fatalf("%+v", p)
	}
	p = ApplyPreset(PresetMedia, DefaultUserPolicy())
	if len(p.Services) != 2 {
		t.Fatalf("%+v", p)
	}
	p = ApplyPreset(PresetNone, DefaultUserPolicy())
	if len(p.Services) != 0 || p.ServicesMode != "list" {
		t.Fatalf("%+v", p)
	}
}
