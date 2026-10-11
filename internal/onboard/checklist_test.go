package onboard

import "testing"

func TestDefaultItems(t *testing.T) {
	if len(defaultItems()) < 1 {
		t.Fatal()
	}
}

func TestLoadTemp(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	_ = Load()
}
