package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/logs"
)

func registerLogsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/logs/schedule", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			s := logs.LoadSchedule()
			writeJSON(w, 200, map[string]any{"schedule": s, "display": logs.FormatSchedule(s)})
			return
		}
		if r.Method == http.MethodPost {
			body := readJSON(r)
			s := logs.LoadSchedule()
			if v, ok := body["hour"].(float64); ok {
				s.Hour = int(v)
			}
			if v, ok := body["minute"].(float64); ok {
				s.Minute = int(v)
			}
			if v, ok := body["utc"].(bool); ok {
				s.UTC = v
			}
			if v, ok := body["keep_hours"].(float64); ok {
				s.KeepHours = int(v)
			}
			if err := logs.SaveSchedule(s); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			audit.Log("session", "logs.schedule", logs.FormatSchedule(s), "")
			writeJSON(w, 200, map[string]any{"ok": true, "schedule": s, "display": logs.FormatSchedule(s)})
			return
		}
		writeJSON(w, 405, map[string]string{"error": "method"})
	})
	mux.HandleFunc("/api/logs/rotate", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		out, err := logs.Rotate()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error(), "out": out})
			return
		}
		audit.Log("session", "logs.rotate", "ok", "")
		writeJSON(w, 200, map[string]any{"ok": true, "out": out})
	})
	mux.HandleFunc("/api/logs/export", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		hours := 1
		if q := r.URL.Query().Get("hours"); q != "" {
			if n, err := strconv.Atoi(q); err == nil {
				hours = n
			}
		}
		path, err := logs.ExportHours(hours)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "logs.export", path, "")
		if r.Header.Get("Accept") == "application/json" || r.URL.Query().Get("meta") == "1" {
			writeJSON(w, 200, map[string]any{"ok": true, "path": path, "name": filepath.Base(path)})
			return
		}
		f, err := os.Open(path)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(path)+"\"")
		_, _ = io.Copy(w, f)
	})
}
