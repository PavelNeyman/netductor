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
	mux.HandleFunc("/api/fleet/overview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		d := fleet.Build()
		writeJSON(w, 200, map[string]any{
			"digest": d,
			"hint":   "primary/secondary/edge summary + backup age + versions",
		})
	})
	mux.HandleFunc("/api/fleet/actions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"actions": fleet.ActionRequired()})
	})
}
