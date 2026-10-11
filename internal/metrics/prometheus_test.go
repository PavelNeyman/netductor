package metrics

import (
	"strings"
	"testing"
)

func TestPrometheusTextSmoke(t *testing.T) {
	s := PrometheusText()
	if !strings.Contains(s, "netductor_up 1") {
		t.Fatalf("missing up metric: %q", s)
	}
	if !strings.Contains(s, "# TYPE netductor_up gauge") {
		t.Fatal("missing type")
	}
}
