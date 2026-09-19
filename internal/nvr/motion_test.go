package nvr

import (
	"testing"
	"time"
)

func TestInMotionWindow(t *testing.T) {
	c := MotionConfig{Enabled: true, Windows: []ScheduleWindow{{Start: "09:00", End: "18:00"}}}
	// 2026-09-19 12:00 local — use fixed UTC then ignore tz
	mid := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	if !InMotionWindow(c, mid) {
		t.Fatal("midday should be in")
	}
	night := time.Date(2026, 9, 19, 22, 0, 0, 0, time.UTC)
	if InMotionWindow(c, night) {
		t.Fatal("night should be out")
	}
	c.Enabled = false
	if !InMotionWindow(c, night) {
		t.Fatal("disabled = always")
	}
}

func TestParseHHMM(t *testing.T) {
	if parseHHMM("09:30") != 9*60+30 {
		t.Fatal(parseHHMM("09:30"))
	}
}
