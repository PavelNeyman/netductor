package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/probes"
	"github.com/PavelNeyman/netductor/internal/relay"
	"github.com/PavelNeyman/netductor/internal/session"
	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/sites"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func buildAPIMux() http.Handler {
	edge.EnsureDefaultTemplate()
	mux := http.NewServeMux()

	registerNodesAPI(mux)
	registerAddonsAPI(mux)
	registerRelayAPI(mux)
	registerSSHHostsAPI(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "service": "netductor", "version": version, "time": time.Now().UTC().Format(time.RFC3339)})
	})

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
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		did := edge.DeviceIDFromAuth(auth)
		if did == "" {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
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
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
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
		writeJSON(w, 405, map[string]string{"error": "method"})
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
			writeJSON(w, 405, map[string]string{"error": "method"})
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
			writeJSON(w, 405, map[string]string{"error": "method"})
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
		if !edge.ValidBootstrap(r.Header.Get("Authorization")) {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		st, dtok, isNew := edge.Enroll(payload)
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
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
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
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		q := r.URL.Query().Get("device_id")
		if q != "" && q != did {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		writeJSON(w, 200, map[string]any{"commands": edge.PollCommands(did)})
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
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		tokenDID := edge.DeviceIDFromAuth(r.Header.Get("Authorization"))
		if tokenDID == "" {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
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
			writeJSON(w, 405, map[string]string{"error": "method"})
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

	// --- operator session ---
	
	mux.HandleFunc("/api/nodes/journal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", 405)
			return
		}
		if !requireSession(w, r) {
			return
		}
		unit := r.URL.Query().Get("unit")
		if unit == "" {
			unit = "sing-box"
		}
		// allowlist
		ok := false
		for _, u := range []string{"sing-box", "netductor-api", "netductor-telegram-bot", "blocky", "netductor-relay-agent"} {
			if u == unit {
				ok = true
				break
			}
		}
		if !ok {
			http.Error(w, "unit", 400)
			return
		}
		id := r.URL.Query().Get("id")
		if id != "" && (strings.HasPrefix(id, "relay-") || strings.Contains(id, "relay")) {
			// queue remote journal; return last known log if this is a refresh
			_ = relay.EnqueueCmd(id, "journal")
			log := ""
			for _, d := range relay.List() {
				if d.ID == id {
					log = d.LastCmdLog
					break
				}
			}
			writeJSON(w, 200, map[string]any{"unit": "remote", "id": id, "queued": true, "log": log, "hint": "re-fetch in ~30s for full journal"})
			return
		}
		out, _ := exec.Command("journalctl", "-u", unit, "-n", "80", "--no-pager", "-o", "short-iso").CombinedOutput()
		writeJSON(w, 200, map[string]any{"unit": unit, "log": string(out)})
	})
	mux.HandleFunc("/api/nodes/restart-service", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		if !requireSession(w, r) {
			return
		}
		var body struct {
			Unit string `json:"unit"`
			ID   string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Unit == "" {
			body.Unit = "sing-box"
		}
		ok := false
		for _, u := range []string{"sing-box", "netductor-api", "blocky", "netductor-telegram-bot"} {
			if u == body.Unit {
				ok = true
				break
			}
		}
		if !ok {
			http.Error(w, "unit", 400)
			return
		}
		// core local only for now; relay uses cmd queue
		if body.ID != "" && (strings.HasPrefix(body.ID, "relay-") || strings.Contains(body.ID, "relay")) {
			_ = relay.EnqueueCmd(body.ID, "restart:"+body.Unit)
			writeJSON(w, 200, map[string]any{"ok": true, "queued": true})
			return
		}
		out, err := exec.Command("systemctl", "restart", body.Unit).CombinedOutput()
		writeJSON(w, 200, map[string]any{"ok": err == nil, "out": string(out)})
	})

	mux.HandleFunc("/api/sni-presets", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		_ = vpn.EnsureSNIPresetsFile()
		writeJSON(w, 200, map[string]any{"presets": vpn.ListSNIPresets(), "active": vpn.ActiveSNI()})
	})
	mux.HandleFunc("/api/sni", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]any{"active": vpn.ActiveSNI(), "presets": vpn.ListSNIPresets()})
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var body struct {
			SNI  string `json:"sni"`
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sniName := body.SNI
		if sniName == "" {
			sniName = body.Name
		}
		for _, p := range vpn.ListSNIPresets() {
			if p.Name == sniName {
				sniName = p.SNI
				break
			}
		}
		if err := vpn.SetActiveSNI(sniName); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		ver := relay.BumpConfigVer()
		writeJSON(w, 200, map[string]any{"ok": true, "active": vpn.ActiveSNI(), "relay_config_ver": ver})
	})
	
	mux.HandleFunc("/api/sites", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			list, err := sites.List()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, 200, map[string]any{"sites": list})
			return
		}
		if r.Method == http.MethodPost {
			var s sites.Site
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			out, err := sites.Upsert(s)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, 200, out)
			return
		}
		http.Error(w, "method", 405)
	})
	mux.HandleFunc("/api/sites/rsc", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := r.URL.Query().Get("id")
		rsc, err := sites.RSCForSite(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, 200, map[string]any{"id": id, "rsc": rsc})
	})

	
	mux.HandleFunc("/api/sites/push-rsc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method", 405)
			}
			return
		}
		var body struct {
			SiteID   string `json:"site_id"`
			Host     string `json:"host"`
			User     string `json:"user"`
			Password string `json:"password"`
			Port     int    `json:"port"`
			RpiLAN   string `json:"rpi_lan"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		rsc, err := sites.RSCForSiteWithGateway(body.SiteID, body.RpiLAN)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := mikrotik.PushRSC(body.Host, body.User, body.Password, nil, rsc, body.Port); err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		exp, ok := session.Expiry(tok)
		if !ok {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "expires_unix": exp})
	})
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, metrics.Collect())
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, metrics.Collect())
	})
	mux.HandleFunc("/api/metrics/history", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		limit := 180
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				limit = n
			}
		}
		writeJSON(w, 200, map[string]any{"points": metrics.History(limit)})
	})



	mux.HandleFunc("/api/probes", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"probes": probes.Run(probes.Load()), "config": probes.Load()})
	})
	mux.HandleFunc("/api/latest", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"metrics": metrics.Collect(), "probes": probes.Run(probes.Load())})
	})
	mux.HandleFunc("/api/probes/uptime", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		limit := 1440
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				limit = n
			}
		}
		writeJSON(w, 200, map[string]any{"uptime": metrics.ProbeUptime(limit)})
	})
	mux.HandleFunc("/api/probes/config", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, probes.Load())
		case http.MethodPost, http.MethodPut:
			cfg := readJSON(r)
			if err := probes.Save(cfg); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			writeJSON(w, 405, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		m := metrics.Collect()
		mm := vpn.CollectMismatch(30)
		relays := []map[string]any{}
		for _, d := range relay.List() {
			relays = append(relays, map[string]any{
				"id": d.ID, "name": d.Name, "ip": d.PublicIP,
				"mismatch_total": d.MismatchTotal, "mismatch_by_ip": d.MismatchByIP,
			})
		}
		writeJSON(w, 200, map[string]any{
			"ok": true, "service": "netductor", "version": version,
			"metrics": m, "probes": probes.Run(probes.Load()),
			"mismatch": mm, "relay_mismatch": relays,
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		m := metrics.Collect()
		mm := vpn.CollectMismatch(30)
		writeJSON(w, 200, map[string]any{"ok": true, "metrics": m, "probes": probes.Run(probes.Load()), "mismatch": mm})
	})

	// --- VPN users (operator session) ---
	mux.HandleFunc("/vpn/users", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			users, err := vpn.List()
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"users": users})
		case http.MethodPost:
			body := readJSON(r)
			name, _ := body["name"].(string)
			note, _ := body["note"].(string)
			name = strings.TrimSpace(name)
			note = strings.TrimSpace(note)
			if !vpn.ValidName(name) {
				writeJSON(w, 400, map[string]string{"error": "bad name"})
				return
			}
			out, err := vpn.Add(name, note)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out), "name": name})
		default:
			writeJSON(w, 405, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/vpn/users/", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/vpn/users/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		name, action := parts[0], parts[1]
		if !vpn.ValidName(name) {
			writeJSON(w, 400, map[string]string{"error": "bad name"})
			return
		}
		switch {
		case action == "qr" && r.Method == http.MethodGet:
			qp := vpn.QRPath(name)
			b, err := os.ReadFile(qp)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": "no qr"})
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(200)
			_, _ = w.Write(b)
		case action == "subscription" && r.Method == http.MethodGet:
			sub, ok := vpn.ReadClient(name, "subscription.txt", "link.txt")
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "not found"})
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(sub + "\n"))
		case action == "link" && r.Method == http.MethodGet:
			sub, ok := vpn.ReadClient(name, "subscription.txt", "link.txt")
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "not found"})
				return
			}
			vless, _ := vpn.ReadClient(name, "link-vless.txt", "link.txt")
			hy2, _ := vpn.ReadClient(name, "link-hy2.txt")
			writeJSON(w, 200, map[string]any{"name": name, "subscription": sub, "vless": vless, "hy2": hy2})
		case action == "note" && r.Method == http.MethodPost:
			note, _ := readJSON(r)["note"].(string)
			out, err := vpn.Note(name, note)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		case action == "disable" && r.Method == http.MethodPost:
			out, err := vpn.Disable(name)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		case action == "enable" && r.Method == http.MethodPost:
			out, err := vpn.Enable(name)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		case action == "revoke" && r.Method == http.MethodPost:
			out, err := vpn.Revoke(name)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		default:
			writeJSON(w, 404, map[string]string{"error": "not found"})
		}
	})

	// --- Admin SPA ---
	root := adminRoot()
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})
	mux.Handle("/admin/", http.StripPrefix("/admin/", http.FileServer(http.Dir(root))))
	return mux
}


func runServe(args []string) {
	bind := envOr("NETDUCTOR_API_BIND", "127.0.0.1")
	port := envOr("NETDUCTOR_API_PORT", "8787")
	tlsCert := envOr("NETDUCTOR_TLS_CERT", "")
	tlsKey := envOr("NETDUCTOR_TLS_KEY", "")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--bind":
			if i+1 < len(args) {
				bind, i = args[i+1], i+1
			}
		case "--port":
			if i+1 < len(args) {
				port, i = args[i+1], i+1
			}
		case "--help", "-h":
			printHelp()
			return
		}
	}
	if bind != "127.0.0.1" && bind != "localhost" {
		if os.Getenv("NETDUCTOR_API_PUBLIC") != "1" {
			fmt.Fprintln(os.Stderr, "refusing non-local API bind without NETDUCTOR_API_PUBLIC=1")
			os.Exit(2)
		}
		if tlsCert == "" || tlsKey == "" {
			fmt.Fprintln(os.Stderr, "public API bind requires --tls-cert and --tls-key (or NETDUCTOR_TLS_*)")
			os.Exit(2)
		}
	}
	mux := buildAPIMux()
	root := adminRoot()
	addr := bind + ":" + port
	fmt.Fprintf(os.Stderr, "netductor serve on http://%s admin=%s\n", addr, root)
	if tlsCert != "" && tlsKey != "" {
		fmt.Fprintf(os.Stderr, "netductor serve TLS on https://%s\n", addr)
		startRelayAgentListener()
		if err := http.ListenAndServeTLS(addr, tlsCert, tlsKey, withSecurity(mux)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	startRelayAgentListener()
	if err := http.ListenAndServe(addr, withSecurity(mux)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
