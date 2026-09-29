package opcatalog

import "testing"

func TestGroupsCoverSections(t *testing.T) {
	covered := map[string]bool{}
	for _, g := range Groups() {
		if g.ID == "" || g.LabelEN == "" {
			t.Fatalf("empty group %+v", g)
		}
		for _, s := range g.Sections {
			covered[s] = true
		}
	}
	for _, a := range All() {
		if !covered[a.Section] {
			t.Errorf("section %q of action %s not in any Group", a.Section, a.ID)
		}
	}
}
