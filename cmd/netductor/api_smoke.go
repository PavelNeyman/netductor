package main

import (
	"net/http"
	"os/exec"
	"strings"
)

func registerSmokeAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/smoke", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		mode := r.URL.Query().Get("mode")
		if mode == "" {
			mode = "dual"
		}
		out, err := exec.Command("netductor", "smoke", mode).CombinedOutput()
		writeJSON(w, 200, map[string]any{"ok": err == nil, "output": strings.TrimSpace(string(out))})
	})
}
