package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/devices"
	"github.com/PavelNeyman/netductor/internal/onboard"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func registerDevicesAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/devices", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		user := strings.TrimSpace(r.URL.Query().Get("user"))
		if r.URL.Query().Get("refresh") == "1" {
			_, _ = devices.RefreshFromJournal(120)
		}
		list := devices.List()
		if user != "" {
			list = devices.ListByUser(user)
		}
		writeJSON(w, 200, map[string]any{"ok": true, "devices": list})
	})
}

func registerIncidentAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/incident/collect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		hours := 1
		body := readJSON(r)
		if v, ok := body["hours"].(float64); ok && v >= 1 && v <= 24 {
			hours = int(v)
		}
		outDir := filepath.Join(paths.StateDir(), "incidents", time.Now().UTC().Format("20060102T150405Z"))
		_ = exec.Command("mkdir", "-p", outDir).Run()
			since := fmt.Sprintf("%d hours ago", hours)
		_ = exec.Command("bash", "-c",
			"journalctl -u sing-box -u netductor-api -u netductor-telegram-bot --since '"+since+"' --no-pager > '"+outDir+"/journal.txt' 2>&1; "+
				"/usr/local/bin/netductor channel status > '"+outDir+"/channel.txt' 2>&1; "+
				"/usr/local/bin/netductor doctor > '"+outDir+"/doctor.txt' 2>&1; "+
				"ss -tulnp > '"+outDir+"/ss.txt' 2>&1").Run()
		tar := outDir + ".tar.gz"
		_ = exec.Command("tar", "-C", filepath.Dir(outDir), "-czf", tar, filepath.Base(outDir)).Run()
		writeJSON(w, 200, map[string]any{"ok": true, "dir": outDir, "archive": tar})
	})
}

func registerOnboardAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/onboard", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{"ok": true, "checklist": onboard.Load()})
		case http.MethodPost:
			body := readJSON(r)
			id, _ := body["id"].(string)
			done, _ := body["done"].(bool)
			if id == "" {
				writeJSON(w, 400, map[string]any{"ok": false, "error": "id required"})
				return
			}
			s, err := onboard.Mark(id, done)
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "checklist": s})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
}
