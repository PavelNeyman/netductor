package channels

import "testing"

func TestSparklineConstant(t *testing.T) {
	s := Sparkline([]float64{5, 5, 5})
	if s == "" || len([]rune(s)) != 3 {
		t.Fatalf("spark %q", s)
	}
}

func TestSparklineVarying(t *testing.T) {
	s := Sparkline([]float64{1, 2, 3, 10})
	if len([]rune(s)) != 4 {
		t.Fatalf("len %q", s)
	}
}

func TestHistorySeriesEmpty(t *testing.T) {
	if len(HistorySeries(nil, "C2", "", "rtt", 10)) != 0 {
		t.Fatal("expected empty")
	}
}
