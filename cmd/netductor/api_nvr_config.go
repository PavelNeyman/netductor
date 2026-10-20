package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/nvr"
)

func registerNVRConfig(mux *http.ServeMux) {
	mux.HandleFunc("/api/nvr/config", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			cfg := nvr.LoadConfig()
			last, ok := nvr.LastRetentionReport()
			writeJSON(w, 200, map[string]any{
				"config":            cfg,
				"recorders":         nvr.RecorderRunning(),
				"last_retention":    last,
				"last_retention_ok": ok,
			})
		case http.MethodPost:
			body := readJSON(r)
			cfg := nvr.LoadConfig()
			if v, ok := body["storage_backend"].(string); ok && v != "" {
				cfg.StorageBackend = v
			}
			if v, ok := body["path"].(string); ok && v != "" {
				cfg.Path = v
			}
			if v, ok := body["segment_sec"].(float64); ok && v > 0 {
				cfg.SegmentSec = int(v)
			}
			if v, ok := body["retention_days"].(float64); ok {
				cfg.RetentionDays = int(v)
			}
			if v, ok := body["max_gb"].(float64); ok {
				cfg.MaxGB = v
			}
			if v, ok := body["min_free_gb"].(float64); ok {
				cfg.MinFreeGB = v
			}
			if v, ok := body["rotate_interval_sec"].(float64); ok && v > 0 {
				cfg.RotateIntervalSec = int(v)
			}
			if _, ok := body["record_enabled"]; ok {
				cfg.RecordEnabled = nvrTruthy(body["record_enabled"], cfg.RecordEnabled)
			}
			if err := nvr.SaveConfig(cfg); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			_ = nvr.EnsureSegmentsDir()
			writeJSON(w, 200, map[string]any{"ok": true, "config": nvr.LoadConfig()})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/nvr/retention/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		rep, err := nvr.RunRetention(nvr.LoadConfig())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "report": rep})
	})

	mux.HandleFunc("/api/nvr/segments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		cfg := nvr.LoadConfig()
		files, err := nvr.ListSegmentFiles(cfg.Path)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		cam := r.URL.Query().Get("camera")
		if cam != "" {
			filtered := files[:0]
			for _, f := range files {
				if f.Camera == cam {
					filtered = append(filtered, f)
				}
			}
			files = filtered
		}
		writeJSON(w, 200, map[string]any{"segments": files, "count": len(files)})
	})

	mux.HandleFunc("/api/nvr/recorder/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		id := nvrStr(body["id"])
		c, ok := nvr.GetCamera(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "camera not found"})
			return
		}
		if err := nvr.StartRecorder(c); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	// Agent (device token) or session may upload a segment file.
}
