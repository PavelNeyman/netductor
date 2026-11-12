package channels

import (
	"testing"
	"time"
)

func TestParseProbeTarget(t *testing.T) {
	h, p := ParseProbeTarget("1.2.3.4:8443")
	if h != "1.2.3.4" || p != 8443 {
		t.Fatalf("%s %d", h, p)
	}
	h, p = ParseProbeTarget("example.com")
	if h != "example.com" || p != 443 {
		t.Fatalf("%s %d", h, p)
	}
}

func TestProbeTCPLocalhostFailsFast(t *testing.T) {
	r := ProbeTCP("127.0.0.1", 1, 200*time.Millisecond)
	if r.OK {
		t.Fatal("expected fail")
	}
}

func TestFormatProbeTextEmpty(t *testing.T) {
	s := FormatProbeText(nil)
	if s == "" {
		t.Fatal("empty")
	}
}
