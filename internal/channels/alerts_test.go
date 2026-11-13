package channels

import "testing"

func TestEvaluateSampleAlertsRTTNeedsSustained(t *testing.T) {
	// single spike must not alert
	pt := SamplePoint{Channels: []ChannelSample{{ID: "C2", Peer: "p1", OK: true, RTTMs: 1200}}}
	if al := EvaluateSampleAlerts(pt, nil); len(al) != 0 {
		t.Fatalf("single spike must not alert: %v", al)
	}
}

func TestEvaluateSampleAlertsRTTSustainedCrit(t *testing.T) {
	hist := []SamplePoint{
		{Channels: []ChannelSample{{ID: "C2", Peer: "p1", OK: true, RTTMs: 1100}}},
		{Channels: []ChannelSample{{ID: "C2", Peer: "p1", OK: true, RTTMs: 1050}}},
	}
	pt := SamplePoint{Channels: []ChannelSample{{ID: "C2", Peer: "p1", OK: true, RTTMs: 1200}}}
	al := EvaluateSampleAlerts(pt, hist)
	if len(al) == 0 {
		t.Fatal("expected sustained uplink RTT crit")
	}
	if !al[0].Incident {
		t.Fatal("crit should request incident")
	}
}

func TestEvaluateSampleAlertsC3OneSecondNoAlert(t *testing.T) {
	// ~1s face dial is normal under Reality accept — must not page
	hist := []SamplePoint{
		{Channels: []ChannelSample{{ID: "C3", Peer: "p1", OK: true, RTTMs: 1050}}},
		{Channels: []ChannelSample{{ID: "C3", Peer: "p1", OK: true, RTTMs: 1080}}},
	}
	pt := SamplePoint{Channels: []ChannelSample{{ID: "C3", Peer: "p1", OK: true, RTTMs: 1070}}}
	if al := EvaluateSampleAlerts(pt, hist); len(al) != 0 {
		t.Fatalf("C3 ~1s must not alert: %v", al)
	}
}

func TestEvaluateSampleAlertsIdleRateNoAlert(t *testing.T) {
	pt := SamplePoint{Channels: []ChannelSample{{ID: "C7", OK: true, RxBps: 0, TxBps: 0}}}
	if al := EvaluateSampleAlerts(pt, nil); len(al) != 0 {
		t.Fatalf("idle must not alert: %v", al)
	}
}
