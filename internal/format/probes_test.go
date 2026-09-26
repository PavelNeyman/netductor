package format

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatProbes(t *testing.T) {
	raw := []byte(`{"probes":[{"name":"api","ok":true,"ms":1.2},{"name":"vless","ok":false,"ms":5,"error":"refused"}]}`)
	var v any
	_ = json.Unmarshal(raw, &v)
	r := formatProbes(v, true)
	if !strings.Contains(r.HTML, "Пробы") || !strings.Contains(r.HTML, "api") {
		t.Fatalf("%s", r.HTML)
	}
}
