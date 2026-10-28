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
		// Recovery site binding:
		// - approved device must already have the same site_id (empty ≠ match)
		// - new/pending may receive site_id on first bind
		if reserveID != "" {
			if did, _ := payload["device_id"].(string); did != "" {
				sid, _ := payload["site_id"].(string)
				sid = strings.TrimSpace(sid)
				st0 := edge.DeviceStatus(did)
				if st0 == edge.StatusApproved {
					existing := edge.DeviceSiteID(did)
					if existing == "" || sid == "" || existing != sid {
						edge.AbortRecoveryCode(raw, reserveID)
						writeJSON(w, http.StatusForbidden, map[string]string{"error": "site_mismatch"})
						return
					}
				}
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
			if did, _ := payload["device_id"].(string); did != "" {
				if sid, _ := payload["site_id"].(string); sid != "" {
					if err := edge.SetSiteID(did, sid); err != nil {
						edge.AbortRecoveryCode(raw, reserveID)
						writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "site_bind_failed"})
						return
					}
				}
			}
			// Single durable transition → UsedAt after site bind succeeds.
			if !edge.FinalizeRecoveryAfterEnroll(raw, reserveID) {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": "recovery_finalize_failed", "status": st,
				})
				return
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
		// Recovery may claim armed TokenPickup only when site checks passed.
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
