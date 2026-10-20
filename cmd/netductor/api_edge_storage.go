package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/edgestorage"
)

func registerEdgeStorage(mux *http.ServeMux) {
	mux.HandleFunc("/api/edge/storage/mute", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		body := readJSON(r)
		id, _ := body["device_id"].(string)
		mute, _ := body["mute"].(bool)
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		if err := edge.SetUSBAlertMute(id, mute); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "mute": mute})
	})
	mux.HandleFunc("/api/edge/storage/backups", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := r.URL.Query().Get("device_id")
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		list, err := edgestorage.ListManifests(id)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		uploads := edge.ListBackups(id)
		writeJSON(w, 200, map[string]any{"ok": true, "backups": list, "uploads": uploads})
	})
	mux.HandleFunc("/api/edge/storage/backup-push", func(w http.ResponseWriter, r *http.Request) {
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
		kind := r.URL.Query().Get("kind")
		if kind != "A" && kind != "B" {
			kind = "B"
		}
		did := r.URL.Query().Get("device_id")
		if did == "" {
			did = tokenDID
		}
		if did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		dir := edgestorage.Dir(did)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		name := "backup-" + kind + "-" + time.Now().UTC().Format("20060102T150405Z") + ".tar.gz"
		path := filepath.Join(dir, name)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		n, err := io.Copy(f, io.LimitReader(r.Body, 512<<20))
		_ = f.Close()
		if err != nil {
			_ = os.Remove(path)
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		if err := edgestorage.SaveManifest(did, edgestorage.Manifest{Kind: kind, File: name, Note: fmt.Sprintf("%d bytes", n)}); err != nil {
			_ = os.Remove(path)
			writeJSON(w, 500, map[string]string{"error": "manifest: " + err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "file": name, "bytes": n})
	})

	mux.HandleFunc("/api/edge/storage/latest-b", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" && !requireSession(w, r) {
			return
		}
		did := r.URL.Query().Get("device_id")
		if did == "" {
			did = tokenDID
		}
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		if tokenDID != "" && did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		_, name, err := edgestorage.LatestB(did)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "file": name})
	})
	mux.HandleFunc("/api/edge/storage/download", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" && !requireSession(w, r) {
			return
		}
		did := r.URL.Query().Get("device_id")
		name := filepath.Base(r.URL.Query().Get("name"))
		if did == "" {
			did = tokenDID
		}
		if did == "" || name == "" || name == "." || strings.Contains(name, "..") {
			writeJSON(w, 400, map[string]string{"error": "device_id and name"})
			return
		}
		if tokenDID != "" && did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		path := filepath.Join(edgestorage.Dir(did), name)
		b, err := os.ReadFile(path)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
		w.WriteHeader(200)
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/edge/storage/restore", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		body := readJSON(r)
		id, _ := body["device_id"].(string)
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		format, _ := body["format"].(bool)
		if format {
			_ = edge.EnqueueCmd(id, "storage_init_usb", "format=yes")
		}
		arg, _ := body["file"].(string)
		cmdID := edge.EnqueueCmd(id, "storage_restore_remote", arg)
		if cmdID == "" {
			writeJSON(w, 400, map[string]string{"error": "enqueue failed (device not approved?)"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": cmdID, "format": format})
	})
	mux.HandleFunc("/api/edge/storage/backup-now", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		body := readJSON(r)
		id, _ := body["device_id"].(string)
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		idA := edge.EnqueueCmd(id, "config_backup", "")
		idB := edge.EnqueueCmd(id, "storage_backup", "")
		writeJSON(w, 200, map[string]any{"ok": true, "config_backup": idA, "storage_backup": idB})
	})
	mux.HandleFunc("/api/edge/storage/offline-pack", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		body := readJSON(r)
		id, _ := body["device_id"].(string)
		dest, _ := body["dest"].(string)
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id"})
			return
		}
		if dest == "" {
			dest = filepath.Join("/var/lib/netductor/edge-offline", id)
		}
		dir, err := edgestorage.WriteOfflinePack(id, dest)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "path": dir})
	})

}
