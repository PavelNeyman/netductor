package upstream

import "testing"

func TestDefaultPins(t *testing.T) {
	if len(defaultPins()) < 1 {
		t.Fatal()
	}
}

func TestLoadFallsBack(t *testing.T) {
	t.Setenv("NETDUCTOR_ETC", t.TempDir())
	if len(Load()) < 1 {
		t.Fatal()
	}
}
