package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/PavelNeyman/netductor/internal/edge"
)

func registerEdgeMeta(mux *http.ServeMux) {
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
			w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentFilename(name)+"\"")
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
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		did := edge.DeviceIDFromAuth(auth)
		if did == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
}
