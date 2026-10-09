package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerCanaryAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/canary", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			list := vpn.LoadCanaryUsers()
			st10 := vpn.CollectMismatch(10)
			st30 := vpn.CollectMismatch(30)
			writeJSON(w, 200, map[string]any{
				"ok":    true,
				"users": list,
				"mismatch_10m": map[string]any{"total": st10.Total, "canary": st10.CanaryTotal, "other": st10.OtherTotal},
				"mismatch_30m": map[string]any{"total": st30.Total, "canary": st30.CanaryTotal, "other": st30.OtherTotal},
			})
		case http.MethodPost:
			body := readJSON(r)
			if name, ok := body["toggle"].(string); ok && strings.TrimSpace(name) != "" {
				on, err := vpn.ToggleCanary(strings.TrimSpace(name))
				if err != nil {
					writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
					return
				}
				audit.Log("session", "canary.toggle", name, fmt.Sprintf("on=%v", on))
				writeJSON(w, 200, map[string]any{"ok": true, "name": name, "canary": on, "users": vpn.LoadCanaryUsers()})
				return
			}
			if raw, ok := body["users"].([]any); ok {
				var names []string
				for _, v := range raw {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						names = append(names, strings.TrimSpace(s))
					}
				}
				if err := vpn.SaveCanaryUsers(names); err != nil {
					writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
					return
				}
				audit.Log("session", "canary.set", "", "")
				writeJSON(w, 200, map[string]any{"ok": true, "users": names})
				return
			}
			writeJSON(w, 400, map[string]any{"ok": false, "error": "toggle or users required"})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
}
