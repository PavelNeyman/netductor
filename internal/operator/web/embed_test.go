package web

import (
	"strings"
	"testing"
)

func TestEmbedPresent(t *testing.T) {
	for _, name := range []string{"index.html", "app.css", "app.js"} {
		b, err := FS.ReadFile(name)
		if err != nil || len(b) < 20 {
			t.Fatalf("%s: %v len=%d", name, err, len(b))
		}
	}
	js, _ := FS.ReadFile("app.js")
	if !strings.Contains(string(js), "/api/edge/templates/vpn") {
		t.Fatal("op web must include edge templates/vpn API")
	}
	if !strings.Contains(string(js), "stack") {
		// stack controls are part of current op UI surface
		t.Log("note: stack string not found in app.js (ok if renamed)")
	}
}

func TestNoLegacyAdminTree(t *testing.T) {
	// architectural guard: product UI lives only in this package
	if _, err := FS.ReadFile("style.css"); err == nil {
		t.Fatal("legacy style.css name should not be in op embed")
	}
}
