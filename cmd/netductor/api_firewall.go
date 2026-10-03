package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/PavelNeyman/netductor/internal/firewall"
)

func registerFirewallAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/firewall/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		st := firewall.Collect("")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	})
	mux.HandleFunc("/api/firewall/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		role := r.URL.Query().Get("role")
		if err := firewall.ApplyRole(role); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st := firewall.Collect(role)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	})
	mux.HandleFunc("/api/firewall/heal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		force := r.URL.Query().Get("force") == "1" || os.Getenv("NETDUCTOR_FW_AUTOHEAL") == "1"
		healed, err := firewall.HealIfNeeded("", force)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"healed": healed, "status": firewall.Collect("")})
	})
}
