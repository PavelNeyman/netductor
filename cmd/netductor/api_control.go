package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/controlplane"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/session"
)

func registerControlAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/control/backup/latest", handleControlBackupLatest)
	mux.HandleFunc("/api/control/status", handleControlStatus)
}

func handleControlStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	// readable without session for probes; no secrets
	writeJSON(w, 200, map[string]any{
		"role":          controlplane.Get(),
		"write_allowed": controlplane.WriteAllowed(),
	})
}

func handleControlBackupLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	if !controlplane.WriteAllowed() {
		// Standby must not serve as SoT source
		writeJSON(w, 503, map[string]string{"error": "control_standby", "message": "not the active control"})
		return
	}
	if !authorizeControlBackupPull(w, r) {
		return
	}
	dir := filepath.Join(paths.StateDir(), "backups")
	ents, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "no backups", 404)
		return
	}
	var latest string
	var latestT time.Time
	for _, e := range ents {
		n := e.Name()
		if !strings.HasSuffix(n, ".ndenc") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if latest == "" || info.ModTime().After(latestT) {
			latestT = info.ModTime()
			latest = filepath.Join(dir, n)
		}
	}
	if latest == "" {
		http.Error(w, "no backups", 404)
		return
	}
	f, err := os.Open(latest)
	if err != nil {
		http.Error(w, "backup unavailable", 500)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.Error(w, "backup unavailable", 500)
		return
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		http.Error(w, "backup read failed", 500)
		return
	}
	if _, err := f.Seek(0, 0); err != nil {
		http.Error(w, "backup read failed", 500)
		return
	}
	if b, err := os.ReadFile(filepath.Join(dir, "COMPONENTS.txt")); err == nil {
		w.Header().Set("X-Netductor-Components", strings.ReplaceAll(strings.TrimSpace(string(b)), "\n", ","))
	}
	name := filepath.Base(latest)
	w.Header().Set("X-Netductor-Backup-Name", name)
	w.Header().Set("X-Netductor-Backup-SHA256", hex.EncodeToString(h.Sum(nil)))
	w.Header().Set("X-Netductor-Backup-Size", fmt.Sprintf("%d", st.Size()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentFilename(name)+"\"")
	http.ServeContent(w, r, name, st.ModTime(), f)
}

func authorizeControlBackupPull(w http.ResponseWriter, r *http.Request) bool {
	tok := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if tok == "" {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return false
	}
	if session.Valid(tok) {
		return true
	}
	want, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", "control_standby_token"))
	if err != nil || len(strings.TrimSpace(string(want))) == 0 {
		writeJSON(w, 401, map[string]string{"error": "unauthorized", "message": "no control_standby_token on active"})
		return false
	}
	wb := []byte(strings.TrimSpace(string(want)))
	tb := []byte(tok)
	if len(wb) != len(tb) || subtle.ConstantTimeCompare(wb, tb) != 1 {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return false
	}
	return true
}
