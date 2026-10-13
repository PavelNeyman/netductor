package main

import (
	"encoding/json"
	"net/http"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func registerBuildHostAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/build-host/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "token"})
			return
		}
		var facts map[string]any
		_ = json.NewDecoder(r.Body).Decode(&facts)
		if err := gitstore.RecordBuildHostHeartbeat(token, facts); err != nil {
			writeJSON(w, 401, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/build-host/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		list := gitstore.ListBuildHostStates()
		for i := range list {
			list[i].Token = ""
		}
		writeJSON(w, 200, map[string]any{"ok": true, "hosts": list, "config": gitstore.LoadBuildHost(), "status": gitstore.BuildHostStatusLine()})
	})
		mux.HandleFunc("/api/build-host/deploy", func(w http.ResponseWriter, r *http.Request) {
		// Deploy is operator-only (Mac). Node must not orchestrate build-host.
		writeJSON(w, 410, map[string]any{
			"ok": false,
			"error": "build-host deploy only from operator (netductor-op tui / deploy build-host)",
		})
	})
}
