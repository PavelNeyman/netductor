package format

import "testing"

func TestHealth(t *testing.T) {
	raw := []byte(`{"ok":true,"service":"netductor","version":"0.9.20"}`)
	r := API("health", raw, "en")
	if r.HTML == "" || !contains(r.HTML, "table") {
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

func TestGenericNested(t *testing.T) {
	raw := []byte(`{"ok":true,"metrics":{"cpu":1},"nodes":[{"id":"a"},{"id":"b"}]}`)
	r := API("status", raw, "en")
	if !contains(r.HTML, "Status") {
		t.Fatalf("html=%s", r.HTML)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || stringIndex(s, sub) >= 0)
}
func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
