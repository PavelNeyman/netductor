package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func registerNVRAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/nvr/cameras", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{"cameras": nvr.ListCameras()})
		case http.MethodPost:
			body := readJSON(r)
			c := nvr.Camera{
				ID:        nvrStr(body["id"]),
				SiteID:    nvrStr(body["site_id"]),
				Name:      nvrStr(body["name"]),
				MAC:       nvrStr(body["mac"]),
				LANIP:     nvrStr(body["lan_ip"]),
				RTSPUser:  nvrStr(body["rtsp_user"]),
				RTSPPath:  nvrStr(body["rtsp_path"]),
				SecretRef: nvrStr(body["secret_ref"]),
				Enabled:   nvrTruthy(body["enabled"], true),
				Record:    nvrTruthy(body["record"], true),
				StreamSub: nvrTruthy(body["stream_sub"], false),
			}
			if v, ok := body["rtsp_port"].(float64); ok {
				c.RTSPPort = int(v)
			}
			if feat, ok := body["features"].(map[string]any); ok {
				c.Features = map[string]bool{}
				for k, v := range feat {
					c.Features[k] = nvrTruthy(v, false)
				}
			}
			pass := nvrStr(body["rtsp_password"])
			out, err := nvr.UpsertCamera(c)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			if pass != "" {
				ref := out.SecretRef
				if ref == "" {
					ref = out.ID
					out.SecretRef = ref
					out, _ = nvr.UpsertCamera(out)
				}
				_ = nvr.SetSecret(ref, pass)
			}
			writeJSON(w, 200, map[string]any{"ok": true, "camera": out})
		default:
			writeJSON(w, 405, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/nvr/cameras/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		id := nvrStr(body["id"])
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": nvr.DeleteCamera(id)})
	})

	mux.HandleFunc("/api/nvr/site/leases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := nvrStr(body["device_id"])
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id required"})
			return
		}
		id := edge.EnqueueCmd(did, "dhcp_leases", "")
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "enqueue failed (device not approved?)"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": id, "hint": "poll /api/edge/results"})
	})

	mux.HandleFunc("/api/nvr/site/wifi_clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := nvrStr(body["device_id"])
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id required"})
			return
		}
		id := edge.EnqueueCmd(did, "wifi_clients", "")
		writeJSON(w, 200, map[string]any{"ok": id != "", "cmd_id": id})
	})

	mux.HandleFunc("/api/nvr/site/dhcp_static", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := nvrStr(body["device_id"])
		mac := nvrStr(body["mac"])
		ip := nvrStr(body["ip"])
		name := nvrStr(body["name"])
		if did == "" || mac == "" || ip == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id, mac, ip required"})
			return
		}
		arg := "mac=" + mac + "|ip=" + ip
		if name != "" {
			arg += "|name=" + name
		}
		id := edge.EnqueueCmd(did, "dhcp_static", arg)
		writeJSON(w, 200, map[string]any{"ok": id != "", "cmd_id": id})
	})

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
			writeJSON(w, 405, map[string]string{"error": "method"})
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
	mux.HandleFunc("/api/nvr/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method"})
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
			if err := nvr.SaveMotion(mc); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "motion": nvr.LoadMotion()})
		default:
			writeJSON(w, 405, map[string]string{"error": "method"})
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
		root := nvr.LoadConfig().Path
		if !strings.HasPrefix(path, root) {
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
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		tok := r.URL.Query().Get("token")
		path, ok := nvr.RedeemClipToken(tok)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "invalid or expired token"})
			return
		}
		root := nvr.LoadConfig().Path
		if !strings.HasPrefix(path, root) {
			writeJSON(w, 400, map[string]string{"error": "bad path"})
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", "attachment")
		http.ServeFile(w, r, path)
	})

	
	mux.HandleFunc("/api/nvr/ptz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		id := nvrStr(body["id"])
		dir := nvrStr(body["dir"])
		c, ok := nvr.GetCamera(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "camera not found"})
			return
		}
		pass := nvr.GetSecret(c.SecretRef)
		if pass == "" {
			pass = nvr.GetSecret(c.ID)
		}
		arg := c.LANIP + "|" + c.RTSPUser + "|" + pass + "|" + dir
		cmdID := edge.EnqueueCmd(c.SiteID, "camera_ptz", arg)
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": cmdID, "note": "Tapo C200 ONVIF :2020 best-effort; not HA plugins"})
	})

	mux.HandleFunc("/api/nvr/go2rtc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		path, err := nvr.WriteGo2RTCConfig()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "path": path})
	})

	mux.HandleFunc("/api/nvr/storage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"storage": nvr.GetStorageStatus(), "config": nvr.LoadConfig()})
	})
	mux.HandleFunc("/api/nvr/recorder/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		nvr.StopRecorder(nvrStr(body["id"]))
		writeJSON(w, 200, map[string]any{"ok": true})
	})
}

func nvrStr(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func nvrTruthy(v any, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true") || t == "yes"
	case float64:
		return t != 0
	case nil:
		return def
	default:
		return def
	}
}
