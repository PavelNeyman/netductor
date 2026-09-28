package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func registerSvcPathsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/svc-paths/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, svcpaths.Status())
	})
	mux.HandleFunc("/api/svc-paths/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		msg, err := svcpaths.Apply()
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error(), "msg": msg})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "msg": msg, "status": svcpaths.Status()})
	})
	mux.HandleFunc("/api/svc-paths/failover", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{
				"policy": svcpaths.LoadPolicy(),
				"state":  svcpaths.LoadState(),
			})
		case http.MethodPost:
			raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<16))
			var in struct {
				Enabled            *bool `json:"enabled"`
				UsersToSPOnVLESSDown *bool `json:"users_to_sp_on_vless_down"`
			}
			_ = json.Unmarshal(raw, &in)
			p := svcpaths.LoadPolicy()
			if in.Enabled != nil {
				p.Enabled = *in.Enabled
			}
			if in.UsersToSPOnVLESSDown != nil {
				p.UsersToSPOnVLESSDown = *in.UsersToSPOnVLESSDown
			}
			_ = svcpaths.SavePolicy(p)
			writeJSON(w, 200, map[string]any{"ok": true, "policy": p})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/svc-paths/failover/tick", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		s, sum, err := svcpaths.RunSecondaryCycle()
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error(), "summary": sum})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "summary": sum, "state": s})
	})
	// bootstrap is operator-machine / deploy only — not exposed on public API surfaces intentionally
	_ = strings.TrimSpace
}
