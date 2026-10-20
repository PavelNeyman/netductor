package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func registerNVRLive(mux *http.ServeMux) {
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
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": cmdID, "note": "edge camera_ptz: tapo-go → pytapo → ONVIF"})
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
	mux.HandleFunc("/api/nvr/live", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}
		c, ok := nvr.GetCamera(id)
		if !ok {
			for _, x := range nvr.ListCameras() {
				if x.Name == id || x.ID == id {
					c, ok = x, true
					break
				}
			}
		}
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "camera not found"})
			return
		}
		key := c.ID
		if key == "" {
			key = c.Name
		}
		writeJSON(w, 200, map[string]any{
			"ok":          true,
			"camera":      c.ID,
			"name":        c.Name,
			"stream_path": "/api/nvr/stream?id=" + url.QueryEscape(key),
			"live_path":   "/api/nvr/live?id=" + url.QueryEscape(key),
			"rtsp_hint":   "rtsp://127.0.0.1:8554/" + key,
			"go2rtc_api":  "http://127.0.0.1:1984",
			"note":        "Use stream_path via session tunnel; go2rtc remains localhost-only",
		})
	})
	mux.HandleFunc("/api/nvr/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		src := id
		if c, ok := nvr.GetCamera(id); ok {
			if c.ID != "" {
				src = c.ID
			} else if c.Name != "" {
				src = c.Name
			}
		}
		u := "http://127.0.0.1:1984/api/stream.mp4?src=" + url.QueryEscape(src)
		client := &http.Client{Timeout: 0} // streaming
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "go2rtc unreachable: "+err.Error()+"; generate yaml (nvr go2rtc) and run go2rtc on primary", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		} else {
			w.Header().Set("Content-Type", "video/mp4")
		}
		w.WriteHeader(resp.StatusCode)
		// Node serve WriteTimeout is 60s; live must not inherit it.
		_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
		_, _ = io.Copy(w, resp.Body)
	})
}
