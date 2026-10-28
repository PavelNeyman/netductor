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
		var reserveID string
		// Atomic reserve → enroll → commit (or abort on failure). Prevents concurrent double-use.
		if !edge.ValidBootstrap(auth) {
			siteID, rid, ok := edge.ReserveRecoveryCode(raw)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "recovery_invalid"})
				return
			}
			reserveID = rid
			if payload == nil {
				payload = map[string]any{}
			}
			if siteID != "" {
				payload["site_id"] = siteID
			}
		}
		st, dtok, isNew := edge.Enroll(payload)
		if st == "error" {
			if reserveID != "" {
				edge.AbortRecoveryCode(raw, reserveID)
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "enroll_persist_failed"})
			return
		}
		if reserveID != "" {
			// Single durable transition → UsedAt. No 120s free window after successful Enroll.
			if !edge.FinalizeRecoveryAfterEnroll(raw, reserveID) {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": "recovery_finalize_failed", "status": st, "device_token": dtok,
				})
				return
			}
		}
		// site_id binding only from recovery code path — not from shared bootstrap enroll.
		if reserveID != "" {
			if did, _ := payload["device_id"].(string); did != "" {
				if sid, _ := payload["site_id"].(string); sid != "" {
					_ = edge.SetSiteID(did, sid)
				}
			}
		}
		if isNew && st == edge.StatusPending {
			did, _ := payload["device_id"].(string)
			board, _ := payload["board"].(string)
			wan, _ := payload["wan_ip"].(string)
			_ = notify.Telegram(fmt.Sprintf("⏳ Edge pending: <b>%s</b>\nboard=%s wan=%s", did, board, wan))
		}
		out := map[string]any{"status": st}
		// H-1 / R-05: shared bootstrap never yields a personal device token.
		// Recovery-code path may claim an armed TokenPickup once (rebind / reissue recovery).
		// Operators otherwise install tokens from Approve/Reissue/Rotate offline.
		if reserveID != "" && st == edge.StatusApproved {
			if did, _ := payload["device_id"].(string); did != "" {
				if pt := edge.PeekPickupToken(did); pt != "" {
					out["device_token"] = pt
				}
			}
		} else if dtok != "" && reserveID != "" {
			out["device_token"] = dtok
		}
		writeJSON(w, 200, out)
	})

}
