package cleanup

import "testing"

func TestHumanBytes(t *testing.T) {
	if humanBytes(0) == "" || humanBytes(1024) == "" {
		t.Fatal()
	}
}

func TestFormatAndDryRun(t *testing.T) {
	if FormatText(Report{}) == "" {
		t.Fatal()
	}
	_ = Run(false)
}
