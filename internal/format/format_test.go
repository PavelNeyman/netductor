package format

import "testing"

func TestHealth(t *testing.T) {
	raw := []byte(`{"ok":true,"service":"netductor","time":"2026-09-25T16:51:53Z","version":"0.9.21"}`)
	r := API("health", raw, "en")
	if r.HTML == "" || !contains(r.HTML, "Health") || !contains(r.HTML, "0.9.21") {
		t.Fatalf("html=%s", r.HTML)
	}
}

func TestMetrics(t *testing.T) {
	raw := []byte(`{"hostname":"nd","cpu_pct":12.5,"loadavg":{"1":0.1,"5":0.2,"15":0.3},"mem":{"total":8589934592,"used":2147483648},"disk":{"use_pct":40},"services":{"sing-box":"active","blocky":"active"}}`)
	r := API("metrics", raw, "ru")
	if !contains(r.HTML, "Метрики") || !contains(r.HTML, "sing-box") {
		t.Fatalf("html=%s", r.HTML)
	}
}

func TestMetricsHist(t *testing.T) {
	raw := []byte(`{"points":[{"ts":1,"cpu_pct":1.5,"loadavg":{"1":0.2},"mem":{"total":100,"used":40}}]}`)
	r := API("metrics-hist", raw, "en")
	if !contains(r.HTML, "history") || !contains(r.HTML, "1.5") {
		t.Fatalf("html=%s", r.HTML)
	}
}

func TestDoctor(t *testing.T) {
	raw := []byte(`{"ok":true,"role":"primary","host":"nd","summary":{"ok":10,"fail":1,"warn":0},"checks":[{"id":"tg","status":"fail"}]}`)
	r := API("doctor", raw, "ru")
	if !contains(r.HTML, "Диагностика") || !contains(r.HTML, "tg") {
		t.Fatalf("html=%s", r.HTML)
	}
}

func contains(s, sub string) bool {
	return stringIndex(s, sub) >= 0
}
func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
