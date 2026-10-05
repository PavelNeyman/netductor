package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/fleet"
)

func registerFleetAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/fleet/digest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, fleet.Build())
	})
}
