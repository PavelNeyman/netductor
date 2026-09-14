package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/relay"
)

func registerSSHHostsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/ssh-hosts", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		kind := r.URL.Query().Get("kind")
		switch r.Method {
		case http.MethodGet:
			out := map[string]any{}
			if kind == "" || kind == "mt" || kind == "mikrotik" || kind == "all" {
				out["mikrotik"] = mikrotik.ListKnownHosts()
			}
			if kind == "" || kind == "relay" || kind == "all" {
				out["relay"] = relay.ListSSHHosts()
			}
			writeJSON(w, 200, out)
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				writeJSON(w, 400, map[string]string{"error": "id"})
				return
			}
			k := r.URL.Query().Get("kind")
			var err error
			switch k {
			case "relay":
				err = relay.ForgetSSHHost(id)
			case "mt", "mikrotik":
				err = mikrotik.ForgetKnownHost(id)
			default:
				e1 := mikrotik.ForgetKnownHost(id)
				e2 := relay.ForgetSSHHost(id)
				if e1 != nil && e2 != nil {
					err = e1
				}
			}
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "forgot": id})
		default:
			writeJSON(w, 405, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/api/ssh-hosts/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		kind := r.URL.Query().Get("kind")
		switch kind {
		case "relay":
			_ = relay.ClearSSHHosts()
		case "mt", "mikrotik":
			_ = mikrotik.ClearKnownHosts()
		default:
			_ = mikrotik.ClearKnownHosts()
			_ = relay.ClearSSHHosts()
		}
		writeJSON(w, 200, map[string]any{"ok": true, "cleared": kind})
	})
}
