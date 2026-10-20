package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

func registerSessionNodes(mux *http.ServeMux) {
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

}
