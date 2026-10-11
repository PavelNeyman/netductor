package channels

import (
	"strings"
	"testing"
	"time"
)

func TestFormatPathE2EEmpty(t *testing.T) {
	s := FormatPathE2E(nil)
	if !strings.Contains(s, "path e2e") {
		t.Fatalf("%q", s)
	}
	if !strings.Contains(s, "no secondaries") {
		t.Fatalf("%q", s)
	}
}

func TestFormatPathE2ERow(t *testing.T) {
	list := []PathE2E{{
		SecondaryID: "sec-1", Name: "secondary", Online: true, PathOK: true,
		UplinkOK: true, MgmtOK: true, Face443: true, SSH52222: true, ICMP: true, SingBoxOK: true,
		Note: "ok",
	}}
	s := FormatPathE2E(list)
	if !strings.Contains(s, "secondary") {
		t.Fatalf("%q", s)
	}
	if !strings.Contains(s, "ok") {
		t.Fatalf("%q", s)
	}
}

func TestFormatTextReport(t *testing.T) {
	r := Report{TS: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	s := FormatText(r)
	if !strings.Contains(s, "channel health") {
		t.Fatalf("%q", s)
	}
	if !strings.Contains(s, "secondaries: (none)") {
		t.Fatalf("%q", s)
	}
}
