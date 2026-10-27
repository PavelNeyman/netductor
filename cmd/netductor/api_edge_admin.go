package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/sites"
)

func registerEdgeAdmin(mux *http.ServeMux) {
	mux.HandleFunc("/api/edge/approve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		tok, err := edge.Approve(did)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		_ = notify.Telegram(fmt.Sprintf("✅ Edge approved: <b>%s</b>", did))
		audit.Log("session", "edge.approve", did, "")
		writeJSON(w, 200, map[string]any{"ok": true, "device_id": did, "device_token": tok})
	})
	mux.HandleFunc("/api/edge/token-ack", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		// Personal device token in Authorization — not session, not bootstrap.
		auth := r.Header.Get("Authorization")
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id required"})
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if err := edge.ConfirmTokenPickup(did, raw); err != nil {
			writeJSON(w, 401, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("edge", "token-ack", did, "")
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/edge/reissue-token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		tok, err := edge.ReissueToken(did)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "edge.reissue-token", did, "")
		writeJSON(w, 200, map[string]any{"ok": true, "device_id": did, "device_token": tok, "token_pickup": true})
	})
	mux.HandleFunc("/api/edge/deny", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		if err := edge.Deny(did); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/edge/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		if err := edge.Revoke(did); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/edge/pending", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"pending": edge.ListPending()})
	})
	mux.HandleFunc("/api/edge/sites", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"sites": edge.ListSiteSummaries()})
	})
	mux.HandleFunc("/api/edge/devices", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"devices": edge.ListDevices()})
	})

	mux.HandleFunc("/api/edge/guest/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		did := r.URL.Query().Get("device_id")
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "id": edge.EnqueueCmd(did, "guest_status", "")})
	})
	mux.HandleFunc("/api/edge/guest/grant", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		code, _ := body["code"].(string)
		mac, _ := body["mac"].(string)
		mins := 10
		switch v := body["minutes"].(type) {
		case float64:
			mins = int(v)
		case string:
			fmt.Sscanf(v, "%d", &mins)
		}
		if mins <= 0 {
			mins = 10
		}
		if mins > 24*60 {
			mins = 24 * 60
		}
		key := code
		if key == "" {
			key = mac
		}
		if did == "" || key == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id and code|mac"})
			return
		}
		arg := fmt.Sprintf("%s|%d", key, mins)
		writeJSON(w, 200, map[string]any{"ok": true, "id": edge.EnqueueCmd(did, "guest_grant", arg)})
	})
	mux.HandleFunc("/api/edge/guest/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		mac, _ := body["mac"].(string)
		if did == "" || mac == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id, mac"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "id": edge.EnqueueCmd(did, "guest_revoke", mac)})
	})

	mux.HandleFunc("/api/edge/cmd", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		action, _ := body["action"].(string)
		arg, _ := body["arg"].(string)
		if did == "" || action == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id and action required"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "id": edge.EnqueueCmd(did, action, arg)})
	})

	mux.HandleFunc("/api/edge/recovery", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		site, _ := body["site_id"].(string)
		note, _ := body["note"].(string)
		code, exp, err := edge.IssueRecoveryCode(site, note, 0)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "edge.recovery", site, note)
		writeJSON(w, 200, map[string]any{
			"ok": true, "code": code, "expires": exp.UTC().Format(time.RFC3339),
			"site_id": site, "hint": "Open https://<router-lan>:7879/netductor-recovery on site Wi-Fi (self-signed TLS)",
		})
	})
	mux.HandleFunc("/api/edge/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		site, _ := body["site_id"].(string)
		note, _ := body["note"].(string)
		if err := edge.RegisterPending(did, site, note); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if site != "" {
			_ = sites.AttachEdge(site, did)
		}
		audit.Log("session", "edge.register", did, site)
		writeJSON(w, 200, map[string]any{"ok": true, "device_id": did, "status": "pending"})
	})
	mux.HandleFunc("/api/edge/set-site", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		site, _ := body["site_id"].(string)
		if err := edge.SetSiteID(did, site); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if site != "" {
			_ = sites.AttachEdge(site, did)
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	// Shared plan/card API for all thin UIs (op TUI, Web). No TG deploy.
	mux.HandleFunc("/api/edge/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var req edge.PlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, map[string]string{"error": "bad json"})
			return
		}
		resp, err := edge.ResolvePlan(req)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, resp)
	})
	mux.HandleFunc("/api/edge/facts", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodPost:
			body := readJSON(r)
			did, _ := body["device_id"].(string)
			if did == "" {
				writeJSON(w, 400, map[string]string{"error": "device_id required"})
				return
			}
			raw, _ := json.Marshal(body["facts"])
			if body["facts"] == nil {
				// allow top-level facts fields
				raw, _ = json.Marshal(body)
			}
			var f edge.DeviceFacts
			if err := json.Unmarshal(raw, &f); err != nil {
				writeJSON(w, 400, map[string]string{"error": "bad facts"})
				return
			}
			if err := edge.SetDeviceFacts(did, f); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "card": edge.CardFromFacts(f)})
		case http.MethodGet:
			did := r.URL.Query().Get("id")
			if did == "" {
				did = r.URL.Query().Get("device_id")
			}
			d, ok := edge.GetDevice(did)
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "not found"})
				return
			}
			out := map[string]any{"device_id": did, "facts_at": d.FactsAt}
			if d.Facts != nil {
				out["facts"] = d.Facts
				out["card"] = edge.CardFromFacts(*d.Facts)
			}
			writeJSON(w, 200, out)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/api/edge/card", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		did := r.URL.Query().Get("id")
		if did == "" {
			did = r.URL.Query().Get("device_id")
		}
		d, ok := edge.GetDevice(did)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		out := map[string]any{
			"device_id": did, "board": d.Board, "arch": d.Arch, "hostname": d.Hostname,
			"wan_ip": d.WANIP, "agent": d.Agent, "healthy": d.Healthy, "facts_at": d.FactsAt,
		}
		if d.Facts != nil {
			out["card"] = edge.CardFromFacts(*d.Facts)
			out["facts"] = d.Facts
		}
		writeJSON(w, 200, out)
	})

	mux.HandleFunc("/api/edge/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		b, err := edge.ExportDevices()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/edge/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		replace, _ := body["replace"].(bool)
		raw, _ := json.Marshal(body)
		// prefer nested devices
		if devs, ok := body["devices"]; ok {
			raw, _ = json.Marshal(map[string]any{"devices": devs})
		}
		n, err := edge.ImportDevices(raw, replace)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "imported": n})
	})
}
