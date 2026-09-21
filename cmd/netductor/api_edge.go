package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/sites"
)

func registerEdgeAPI(mux *http.ServeMux) {
	// --- edge (device token) ---
	mux.HandleFunc("/api/edge/backups", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		edgeDID := edge.DeviceIDFromAuth(auth)
		if edgeDID == "" {
			if !requireSession(w, r) {
				return
			}
		}
		did := r.URL.Query().Get("device_id")
		if did == "" {
			did = edgeDID
		}
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		if edgeDID != "" && did != edgeDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		name := r.URL.Query().Get("name")
		if name != "" {
			path, err := edge.BackupPath(did, name)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": "not found"})
				return
			}
			b, err := os.ReadFile(path)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/gzip")
			w.Header().Set("Content-Disposition", "attachment; filename="+name)
			w.WriteHeader(200)
			_, _ = w.Write(b)
			return
		}
		writeJSON(w, 200, map[string]any{"backups": edge.ListBackups(did)})
	})
	mux.HandleFunc("/api/edge/metrics/history", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		did := r.URL.Query().Get("device_id")
		writeJSON(w, 200, map[string]any{"metrics": edge.ListMetricsTail(did, 100)})
	})
	mux.HandleFunc("/api/edge/backup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		did := edge.DeviceIDFromAuth(auth)
		if did == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if hdr := r.Header.Get("X-Device-ID"); hdr != "" && hdr != did {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		path, err := edge.SaveBackup(did, r.Body)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"ok": "true", "path": path})
	})
	mux.HandleFunc("/api/edge/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&payload)
		did, _ := payload["device_id"].(string)
		if did == "" {
			did = tokenDID
		}
		if did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		_ = edge.SaveMetrics(did, payload)
		writeJSON(w, 200, map[string]string{"ok": "true"})
	})
	mux.HandleFunc("/api/edge/template", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		did := r.URL.Query().Get("device_id")
		if did == "" {
			did = edge.DeviceIDFromAuth(auth)
		}
		if r.Method == http.MethodGet {
			if !edge.RequireApproved(auth, did) {
				writeJSON(w, 403, map[string]string{"error": "not_approved"})
				return
			}
			tmpl, err := edge.TemplateWithVPN(did)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, tmpl)
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
	})
	mux.HandleFunc("/api/edge/templates", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		edge.EnsureDefaultTemplate()
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{"templates": edge.ListTemplates()})
		case http.MethodPost:
			body := readJSON(r)
			id, _ := body["id"].(string)
			if id == "" {
				writeJSON(w, 400, map[string]string{"error": "id"})
				return
			}
			if err := edge.SaveTemplate(id, edge.Template(body)); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/api/edge/bind-template", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		tid, _ := body["template_id"].(string)
		ov, _ := body["overlay"].(map[string]any)
		if err := edge.BindTemplate(did, tid, ov); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/edge/enroll", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = r.RemoteAddr
		}
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
		writeJSON(w, 200, map[string]any{"status": st, "device_token": dtok})
	})
	mux.HandleFunc("/api/edge/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		payload := readJSON(r)
		did, _ := payload["device_id"].(string)
		if did == "" {
			did = tokenDID
			payload["device_id"] = did
		}
		if did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		edge.Heartbeat(payload)
		if ser, _ := payload["cert_serial"].(string); ser != "" {
			_ = mtls.ConfirmRotate(did, ser)
		}
		hn, _ := payload["hostname"].(string)
		ip, _ := payload["wan_ip"].(string)
		if ip == "" {
			ip, _ = payload["public_ip"].(string)
		}
		kind, _ := payload["kind"].(string)
		if kind == "" {
			kind = "openwrt"
		}
		role, _ := payload["role"].(string)
		if role == "" {
			role = "edge"
		}
		resp := map[string]any{"ok": true}
		if did != "" {
			n, err := nodes.UpsertFromDevice(nodes.Node{
				ID: did, Hostname: hn, Role: role, Kind: kind, PublicIP: ip, Status: "online",
			})
			if err == nil && n.DesiredHN != "" {
				resp["desired_hostname"] = n.DesiredHN
			}
		}
		writeJSON(w, 200, resp)
	})
	mux.HandleFunc("/api/edge/commands", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		did := edge.DeviceIDFromAuth(auth)
		if did == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		q := r.URL.Query().Get("device_id")
		if q != "" && q != did {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		writeJSON(w, 200, map[string]any{"commands": edge.PollCommands(did)})
	})

	mux.HandleFunc("/api/edge/mtls/material", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		did := edge.DeviceIDFromAuth(r.Header.Get("Authorization"))
		if did == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ca, cert, key, err := mtls.ReadMaterial(did)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		info, _ := mtls.ReadClientInfo(did)
		writeJSON(w, 200, map[string]any{
			"ca_pem": string(ca), "cert_pem": string(cert), "key_pem": string(key),
			"serial": info.Serial, "node_id": did,
		})
	})

	mux.HandleFunc("/api/edge/rsc", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if edge.DeviceIDFromAuth(auth) == "" {
			if !requireSession(w, r) {
				return
			}
		}
		name := r.URL.Query().Get("name")
		if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
			writeJSON(w, 400, map[string]string{"error": "name"})
			return
		}
		b, err := edge.ReadRSC(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/edge/apply_rsc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		script, _ := body["script"].(string)
		if did == "" || script == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id and script required"})
			return
		}
		name, err := edge.SaveRSC(did, script)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		cid := edge.EnqueueCmd(did, "apply_rsc", name)
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": cid, "rsc": name})
	})
	mux.HandleFunc("/api/edge/results", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"results": edge.ListResults()})
	})
	mux.HandleFunc("/api/edge/cmd_result", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		tokenDID := edge.DeviceIDFromAuth(r.Header.Get("Authorization"))
		if tokenDID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		body := readJSON(r)
		if did, _ := body["device_id"].(string); did != "" && did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		body["device_id"] = tokenDID
		edge.CmdResult(body)
		writeJSON(w, 200, map[string]bool{"ok": true})
	})
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
	mux.HandleFunc("/api/edge/devices", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"devices": edge.ListDevices()})
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
			"site_id": site, "hint": "Open http://<router-lan>:7879/netductor-recovery on site Wi-Fi",
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
