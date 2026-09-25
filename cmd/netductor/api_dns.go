package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/dnsblock"
)

func registerDNSAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/dns/lists", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		writeJSON(w, 200, map[string]any{"lists": dnsblock.Catalog()})
	})
	mux.HandleFunc("/api/dns/set", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		body := readJSON(r)
		id, _ := body["id"].(string)
		on, _ := body["enabled"].(bool)
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}
		if err := dnsblock.SetEnabled(id, on); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "dns.set", id, "")
		writeJSON(w, 200, map[string]any{"ok": true, "lists": dnsblock.Catalog()})
	})
	mux.HandleFunc("/api/dns/reload", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		if err := dnsblock.ReloadBlocky(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "dns.reload", "", "")
		writeJSON(w, 200, map[string]any{"ok": true})
	})
}
