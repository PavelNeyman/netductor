package main

import (
	"testing"

	"github.com/PavelNeyman/netductor/internal/opcatalog"
)

func TestDay2CatalogNonEmpty(t *testing.T) {
	gs := opcatalog.Groups()
	if len(gs) < 3 {
		t.Fatalf("groups=%d", len(gs))
	}
	acts := opcatalog.ForSurface(opcatalog.SurfTUI)
	if len(acts) < 5 {
		t.Fatalf("actions=%d", len(acts))
	}
	// Security actions present
	want := map[string]bool{"ssh-keys": false, "reboot-challenge": false}
	for _, a := range acts {
		if _, ok := want[a.ID]; ok {
			want[a.ID] = true
		}
	}
	for id, ok := range want {
		if !ok {
			t.Fatalf("missing action %s", id)
		}
	}
}
