package main

import (
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
