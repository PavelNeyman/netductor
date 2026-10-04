package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/addons"
)

func registerAddonsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/addons", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, addons.ListAddons())
	})
	mux.HandleFunc("/api/addons/lampac", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, addons.CollectLampac())
	})
	mux.HandleFunc("/api/addons/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "all"
		}
		out, err := addons.Update(name)
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error(), "result": out})
			return
		}
		writeJSON(w, 200, out)
	})
}
