package main

import (
	"encoding/json"
	"net/http"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func registerNVREvents(mux *http.ServeMux) {
	mux.HandleFunc("/api/nvr/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		did := edge.DeviceIDFromAuth(auth)
		sessionOK := false
		if did == "" {
			// try session without writing unauthorized body first
			if !requireSession(w, r) {
				return
			}
			sessionOK = true
		} else if !edge.RequireApproved(auth, did) {
			writeJSON(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		_ = sessionOK
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			writeJSON(w, 400, map[string]string{"error": "multipart: " + err.Error()})
			return
		}
		camID := r.FormValue("camera_id")
		if camID == "" {
			camID = r.Header.Get("X-Camera-Id")
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "file required"})
			return
		}
		defer file.Close()
		name := ""
		if hdr != nil {
			name = hdr.Filename
		}
		path, n, err := nvr.IngestSegment(camID, name, file)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		// opportunistic retention if over cap
		go func() { _, _ = nvr.RunRetention(nvr.LoadConfig()) }()
		writeJSON(w, 200, map[string]any{"ok": true, "path": path, "bytes": n, "device_id": did})
	})

	mux.HandleFunc("/api/nvr/motion", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{"motion": nvr.LoadMotion(), "in_window": nvr.InMotionWindow(nvr.LoadMotion(), nvr.Now())})
		case http.MethodPost:
			body := readJSON(r)
			mc := nvr.LoadMotion()
			if _, ok := body["enabled"]; ok {
				mc.Enabled = nvrTruthy(body["enabled"], mc.Enabled)
			}
			if v, ok := body["timezone"].(string); ok {
				mc.Timezone = v
			}
			if _, ok := body["alert_on_segment"]; ok {
				mc.AlertOnSegment = nvrTruthy(body["alert_on_segment"], mc.AlertOnSegment)
			}
			// windows: optional full replace as JSON array in body
			if raw, ok := body["windows"]; ok {
				b, _ := json.Marshal(raw)
				var wins []nvr.ScheduleWindow
				if json.Unmarshal(b, &wins) == nil {
					mc.Windows = wins
				}
			}
			if raw, ok := body["zones"]; ok {
				b, _ := json.Marshal(raw)
				var zones []nvr.MotionZone
				if json.Unmarshal(b, &zones) == nil {
					mc.Zones = zones
				}
			}
			if err := nvr.SaveMotion(mc); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "motion": nvr.LoadMotion()})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/nvr/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		n := 50
		writeJSON(w, 200, map[string]any{"events": nvr.ListEventsTail(n)})
	})

	mux.HandleFunc("/api/nvr/clip/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		path := nvrStr(body["path"])
		cam := nvrStr(body["camera_id"])
		ttl := 120
		if v, ok := body["ttl_sec"].(float64); ok && v > 0 {
			ttl = int(v)
		}
		if path == "" {
			writeJSON(w, 400, map[string]string{"error": "path required"})
			return
		}
		// path must be under segments root
		root := nvr.SegmentsRoot()
		if !nvr.PathUnderRoot(root, path) {
			writeJSON(w, 400, map[string]string{"error": "path outside nvr root"})
			return
		}
		tok, err := nvr.IssueClipToken(cam, path, ttl)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "token": tok, "ttl_sec": ttl, "url": "/api/nvr/clip?token=" + tok})
	})

	mux.HandleFunc("/api/nvr/clip", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		tok := r.URL.Query().Get("token")
		path, ok := nvr.RedeemClipToken(tok)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "invalid or expired token"})
			return
		}
		root := nvr.SegmentsRoot()
		if !nvr.PathUnderRoot(root, path) {
			writeJSON(w, 400, map[string]string{"error": "bad path"})
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", "attachment")
		http.ServeFile(w, r, path)
	})

}
