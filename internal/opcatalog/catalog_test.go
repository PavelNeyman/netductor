package opcatalog

import (
	"strings"
	"testing"
)

func TestAllHaveSurfaceAndUniqueID(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range All() {
		if a.ID == "" || a.Path == "" || a.Method == "" {
			t.Fatalf("incomplete action: %+v", a)
		}
		if seen[a.ID] {
			t.Fatalf("duplicate id %s", a.ID)
		}
		seen[a.ID] = true
		if len(a.Surfaces) == 0 {
			t.Fatalf("%s has no surfaces", a.ID)
		}
		for _, s := range a.Surfaces {
			switch s {
			case SurfWeb, SurfTG, SurfCLI, SurfTUI:
			default:
				t.Fatalf("%s unknown surface %q", a.ID, s)
			}
		}
	}
	if len(seen) < 20 {
		t.Fatalf("expected richer catalog, got %d", len(seen))
	}
}

func TestMatrixMarkdown(t *testing.T) {
	md := MatrixMarkdown()
	if !strings.Contains(md, "| ID |") || !strings.Contains(md, "doctor") {
		t.Fatal("matrix missing expected content")
	}
}

func TestForSurfaceTGHasDoctor(t *testing.T) {
	tg := ForSurface(SurfTG)
	found := false
	for _, a := range tg {
		if a.ID == "doctor" {
			found = true
		}
		if a.ID == "metrics" {
			t.Fatal("metrics should not be on TG (use status)")
		}
	}
	if !found {
		t.Fatal("doctor missing on tg")
	}
}
