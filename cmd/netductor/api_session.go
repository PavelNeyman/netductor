package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/probes"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/session"
	"github.com/PavelNeyman/netductor/internal/sites"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

var allowedSystemdRestart = map[string]bool{
	"netductor-api":            true,
	"netductor-telegram-bot":   true,
	"netductor-secondary-agent": true,
	"netductor-agent":          true,
	"netductor":                true,
}

func allowSystemdUnit(unit string) bool {
	unit = strings.TrimSpace(unit)
	if unit == "" || strings.ContainsAny(unit, " \t\r\n;/|&") {
		return false
	}
	if allowedSystemdRestart[unit] {
		return true
	}
	// allow netductor-*.service style
	if strings.HasPrefix(unit, "netductor-") && !strings.Contains(unit, "..") {
		return true
	}
	return false
}


func registerSessionAPI(mux *http.ServeMux) {
	// --- operator session ---

	mux.HandleFunc("/api/nodes/journal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
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
		for _, u := range []string{"sing-box", "netductor-api", "netductor-telegram-bot", "blocky", "netductor-secondary-agent", "netductor-secondary-agent"} {
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
		if id != "" && (strings.Contains(id, "secondary")) {
			// queue remote journal; return last known log if this is a refresh
			_ = secondary.EnqueueCmd(id, "journal")
			log := ""
			for _, d := range secondary.List() {
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
			http.Error(w, "method", http.StatusMethodNotAllowed)
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
		// core local only for now; secondary uses cmd queue
		if body.ID != "" && (strings.HasPrefix(body.ID, "secondary-") || strings.Contains(body.ID, "secondary")) {
			_ = secondary.EnqueueCmd(body.ID, "restart:"+body.Unit)
			writeJSON(w, 200, map[string]any{"ok": true, "queued": true})
			return
		}
		if !allowSystemdUnit(body.Unit) {
			writeJSON(w, 400, map[string]string{"error": "unit not allowed"})
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
			http.Error(w, "method", http.StatusMethodNotAllowed)
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
		ver := secondary.BumpConfigVer()
		writeJSON(w, 200, map[string]any{"ok": true, "active": vpn.ActiveSNI(), "secondary_config_ver": ver})
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
		http.Error(w, "method", http.StatusMethodNotAllowed)
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
				http.Error(w, "method", http.StatusMethodNotAllowed)
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
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		exp, ok := session.Expiry(tok)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		m := metrics.Collect()
		mm := vpn.CollectMismatch(30)
		relays := []map[string]any{}
		for _, d := range secondary.List() {
			relays = append(relays, map[string]any{
				"id": d.ID, "name": d.Name, "ip": d.PublicIP,
				"mismatch_total": d.MismatchTotal, "mismatch_by_ip": d.MismatchByIP,
			})
		}
		writeJSON(w, 200, map[string]any{
			"ok": true, "service": "netductor", "version": version,
			"metrics": m, "probes": probes.Run(probes.Load()),
			"mismatch": mm, "secondary_mismatch": relays,
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

	mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"events": audit.Tail(100)})
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]any{"sessions": session.List()})
			return
		}
		if r.Method == http.MethodDelete || (r.Method == http.MethodPost && r.URL.Query().Get("action") == "revoke-all") {
			_ = session.RevokeAll()
			audit.Log("session", "session.revoke_all", "", "")
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/mtls/certs", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		clients, _ := mtls.ListClientCerts()
		writeJSON(w, 200, map[string]any{
			"plane":   mtls.ListPlaneCerts(),
			"clients": clients,
			"revoked": func() any { l, _ := mtls.ListRevoked(); return l }(),
		})
	})
	mux.HandleFunc("/api/mtls/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		node, _ := body["node_id"].(string)
		serial, _ := body["serial"].(string)
		reason, _ := body["reason"].(string)
		if reason == "" {
			reason = "api"
		}
		if node != "" {
			ser, err := mtls.RevokeNodeCert(node, reason)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			audit.Log("session", "mtls.revoke", node, ser)
			writeJSON(w, 200, map[string]any{"ok": true, "serial": ser})
			return
		}
		if serial == "" {
			writeJSON(w, 400, map[string]string{"error": "node_id or serial required"})
			return
		}
		if err := mtls.RevokeSerial(serial, "", reason); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "mtls.revoke", serial, reason)
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/mtls/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		node, _ := body["node_id"].(string)
		if node == "" {
			writeJSON(w, 400, map[string]string{"error": "node_id required"})
			return
		}
		info, err := mtls.RotateClientFor(node, "api-rotate")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "mtls.rotate", node, info.Serial)
		writeJSON(w, 200, map[string]any{"ok": true, "cert": info, "path": mtls.ClientDir(node)})
	})

}
