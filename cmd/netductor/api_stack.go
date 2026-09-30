package main

import (
	"encoding/json"
	"net/http"

	"github.com/PavelNeyman/netductor/internal/stack"
)

func registerStackAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/stack/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, stack.Collect())
	})
	mux.HandleFunc("/api/stack/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Version string `json:"version"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := stack.Apply(body.Version); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "status": stack.Collect()})
	})
	mux.HandleFunc("/api/stack/rollback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := stack.Rollback(); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "status": stack.Collect()})
	})
	mux.HandleFunc("/api/stack/watchdog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		stack.WatchdogOnce()
		writeJSON(w, 200, map[string]any{"ok": true, "status": stack.Collect()})
	})
}
