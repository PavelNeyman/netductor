package channels

import "testing"

func TestEvaluateSampleAlertsRTT(t *testing.T) {
	hist := []SamplePoint{{
		Channels: []ChannelSample{{ID: "C2", Peer: "p1", OK: true, RTTMs: 45}},
	}}
	pt := SamplePoint{Channels: []ChannelSample{{ID: "C2", Peer: "p1", OK: true, RTTMs: 320}}}
	al := EvaluateSampleAlerts(pt, hist)
	if len(al) == 0 {
		t.Fatal("expected rtt crit alert")
	}
}

func TestEvaluateSampleAlertsIdleRateNoAlert(t *testing.T) {
	pt := SamplePoint{Channels: []ChannelSample{{ID: "C7", OK: true, RxBps: 0, TxBps: 0}}}
	if al := EvaluateSampleAlerts(pt, nil); len(al) != 0 {
		t.Fatalf("idle must not alert: %v", al)
	}
}
