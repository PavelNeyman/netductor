package upgrade

import "testing"

func TestResolveUpgradePathDirect(t *testing.T) {
	hist := []Meta{
		{Version: "0.9.600"},
		{Version: "0.9.700", MinFrom: "0.9.500"},
	}
	steps, err := ResolveUpgradePath("0.9.550", "0.9.700", hist)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 || steps[0].Tag != "0.9.700" {
		t.Fatalf("steps=%v", steps)
	}
}

// BUG-1: intermediate min_from must be respected.
func TestResolveUpgradePathNestedIntermediateFloors(t *testing.T) {
	hist := []Meta{
		{Version: "0.9.500"},
		{Version: "0.9.550", MinFrom: "0.9.500"},
		{Version: "0.9.600", MinFrom: "0.9.550"},
		{Version: "0.9.700", MinFrom: "0.9.600", Reasons: []string{"slot-material"}},
	}
	steps, err := ResolveUpgradePath("0.9.400", "0.9.700", hist)
	if err != nil {
		t.Fatal(err)
	}
	got := FormatPathCLI(steps)
	if len(steps) < 3 {
		t.Fatalf("expected multi-hop with nested floors, got %s", got)
	}
	if steps[0].Tag == "0.9.600" {
		t.Fatalf("BUG-1: jumped to 600 from 400: %s", got)
	}
	if steps[len(steps)-1].Tag != "0.9.700" {
		t.Fatalf("last=%v path=%s", steps, got)
	}
	cur := "0.9.400"
	for _, s := range steps {
		if err := CheckLocalAllowed(cur, s.MinFrom); err != nil {
			t.Fatalf("hop %s not installable from %s: %v (path %s)", s.Tag, cur, err, got)
		}
		cur = s.Tag
	}
}

func TestResolveUpgradePathNoInstallableIntermediate(t *testing.T) {
	hist := []Meta{
		{Version: "0.9.600", MinFrom: "0.9.550"},
		{Version: "0.9.700", MinFrom: "0.9.600"},
	}
	_, err := ResolveUpgradePath("0.9.400", "0.9.700", hist)
	if err == nil {
		t.Fatal("expected error when no intermediate is installable from 400")
	}
}

func TestResolveUpgradePathUnknownTarget(t *testing.T) {
	steps, err := ResolveUpgradePath("0.9.1", "0.9.999", nil)
	if err != nil || len(steps) != 1 || steps[0].Tag != "0.9.999" {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}
