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
		mode := strings.TrimSpace(r.URL.Query().Get("mode"))
		if mode == "" {
			mode = "dual"
		}
		switch mode {
		case "dual", "secondary", "primary", "local":
		default:
			writeJSON(w, 400, map[string]any{"ok": false, "error": "mode must be dual|secondary|primary|local"})
			return
		}
		out, err := exec.Command("netductor", "smoke", mode).CombinedOutput()
		writeJSON(w, 200, map[string]any{"ok": err == nil, "mode": mode, "output": strings.TrimSpace(string(out))})
	})
}
