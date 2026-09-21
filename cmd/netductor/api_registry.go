package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/registry"
)

func registerRegistryAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/registry/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, registry.StatusInfo())
	})
	mux.HandleFunc("/api/registry/ensure", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		st, err := registry.Ensure()
		audit.Log("session", "registry.ensure", st.Addr, "")
		if err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error(), "status": st})
			return
		}
		writeJSON(w, 200, st)
	})
	mux.HandleFunc("/api/registry/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		err := registry.Stop()
		audit.Log("session", "registry.stop", "", "")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/registry/crane", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		p, err := registry.EnsureCrane()
		audit.Log("session", "registry.crane", p, "")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "path": p})
	})
	mux.HandleFunc("/api/registry/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		if clear, _ := body["clear"].(bool); clear {
			err := registry.ClearAuth()
			audit.Log("session", "registry.auth-clear", "", "")
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		user, _ := body["user"].(string)
		pass, _ := body["password"].(string)
		err := registry.SetAuth(user, pass)
		audit.Log("session", "registry.auth-set", user, "")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/registry/catalog", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		list, err := registry.CatalogDetail()
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"repositories": list})
	})
}
