package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/notify"
)

func registerEdgeEnroll(mux *http.ServeMux) {
	mux.HandleFunc("/api/edge/enroll", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		ip := clientIP(r)
		if !edge.AllowEnroll(ip) {
			writeJSON(w, 429, map[string]string{"error": "rate_limited"})
			return
		}
		auth := r.Header.Get("Authorization")
		if !edge.ValidRecoveryOrBootstrap(auth) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if siteID, ok := edge.ConsumeRecoveryCode(raw); ok && siteID != "" {
			if payload == nil {
				payload = map[string]any{}
			}
			payload["site_id"] = siteID
		}
		st, dtok, isNew := edge.Enroll(payload)
		if did, _ := payload["device_id"].(string); did != "" {
			if sid, _ := payload["site_id"].(string); sid != "" {
				_ = edge.SetSiteID(did, sid)
			}
		}
		if isNew && st == edge.StatusPending {
			did, _ := payload["device_id"].(string)
			board, _ := payload["board"].(string)
			wan, _ := payload["wan_ip"].(string)
			_ = notify.Telegram(fmt.Sprintf("⏳ Edge pending: <b>%s</b>\nboard=%s wan=%s", did, board, wan))
		}
		out := map[string]any{"status": st}
		// H-1: never echo an existing personal token under bootstrap/recovery.
		if dtok != "" {
			out["device_token"] = dtok
		}
		writeJSON(w, 200, out)
	})

}
