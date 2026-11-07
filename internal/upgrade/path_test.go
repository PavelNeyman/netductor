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

func TestResolveUpgradePathNeedsIntermediate(t *testing.T) {
	hist := []Meta{
		{Version: "0.9.400"},
		{Version: "0.9.500", MinFrom: ""},
		{Version: "0.9.600", MinFrom: "0.9.500"},
		{Version: "0.9.700", MinFrom: "0.9.600", Reasons: []string{"slot-material"}},
	}
	steps, err := ResolveUpgradePath("0.9.400", "0.9.700", hist)
	if err != nil {
		t.Fatal(err)
	}
	// 400 < 600 min_from of 700 → hop to 600 (oldest >= 600? min_from 600 → hop 0.9.600), then 700
	if len(steps) < 2 {
		t.Fatalf("expected multi-hop, got %v", steps)
	}
	if steps[len(steps)-1].Tag != "0.9.700" {
		t.Fatalf("last=%v", steps)
	}
	if got := FormatPathCLI(steps); got == "" {
		t.Fatal("empty format")
	}
}

func TestResolveUpgradePathNoIntermediate(t *testing.T) {
	hist := []Meta{
		{Version: "0.9.700", MinFrom: "0.9.650"},
	}
	_, err := ResolveUpgradePath("0.9.400", "0.9.700", hist)
	if err == nil {
		t.Fatal("expected error when history has no intermediate")
	}
}

func TestResolveUpgradePathUnknownTarget(t *testing.T) {
	steps, err := ResolveUpgradePath("0.9.1", "0.9.999", nil)
	if err != nil || len(steps) != 1 || steps[0].Tag != "0.9.999" {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}
