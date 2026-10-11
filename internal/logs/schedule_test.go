package logs

import "testing"

func TestDefaultSchedule(t *testing.T) {
	if DefaultSchedule().KeepHours < 1 {
		t.Fatal("keep")
	}
}

func TestFormatSchedule(t *testing.T) {
	s := Schedule{Hour: 1, Minute: 0, UTC: true, KeepHours: 24}
	if len(FormatSchedule(s)) < 5 {
		t.Fatal(FormatSchedule(s))
	}
}
