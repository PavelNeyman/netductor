package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/cleanup"
)

func registerCleanupAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/cleanup", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, cleanup.Run(false))
		case http.MethodPost:
			body := readJSON(r)
			apply, _ := body["apply"].(bool)
			writeJSON(w, 200, cleanup.Run(apply))
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})
}
