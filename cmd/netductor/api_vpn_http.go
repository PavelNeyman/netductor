package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerVPNHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/api/vpn/refresh-links", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		n, err := vpn.RefreshLinks("")
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "refreshed": n, "error": err.Error()})
			return
		}
		audit.Log("session", "vpn.refresh-links", "", fmt.Sprintf("%d", n))
		writeJSON(w, 200, map[string]any{"ok": true, "refreshed": n})
	})

	// --- backup peer ---
	mux.HandleFunc("/api/backup/peer", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]any{"status": install.BackupPeerStatus()})
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		body := readJSON(r)
		target, _ := body["target"].(string)
		opts, _ := body["opts"].(string)
		if err := install.SetBackupPeer(target, opts); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "status": install.BackupPeerStatus()})
	})
	mux.HandleFunc("/api/backup/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		path, err := install.Backup()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "path": path})
	})

	// --- VPN users (operator session) ---
	
	mux.HandleFunc("/api/backup/schedule", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			s := install.LoadBackupSchedule()
			writeJSON(w, 200, map[string]any{
				"schedule": s,
				"display":  install.FormatBackupSchedule(s),
				"keep":     install.BackupKeepCount(),
			})
			return
		}
		if r.Method == http.MethodPost {
			body := readJSON(r)
			s := install.LoadBackupSchedule()
			if v, ok := body["hour"].(float64); ok {
				s.Hour = int(v)
			}
			if v, ok := body["minute"].(float64); ok {
				s.Minute = int(v)
			}
			if v, ok := body["utc"].(bool); ok {
				s.UTC = v
			}
			if err := install.SaveBackupSchedule(s); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "schedule": s, "display": install.FormatBackupSchedule(s)})
			return
		}
		writeJSON(w, 405, map[string]string{"error": "method"})
	})
	mux.HandleFunc("/api/backup/list", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		writeJSON(w, 200, map[string]any{"files": install.ListBackupFiles()})
	})

	mux.HandleFunc("/vpn/users", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			users, err := vpn.List()
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"users": users})
		case http.MethodPost:
			body := readJSON(r)
			name, _ := body["name"].(string)
			note, _ := body["note"].(string)
			name = strings.TrimSpace(name)
			note = strings.TrimSpace(note)
			if !vpn.ValidName(name) {
				writeJSON(w, 400, map[string]string{"error": "bad name"})
				return
			}
			out, err := vpn.Add(name, note)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out), "name": name})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})
	mux.HandleFunc("/vpn/users/", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/vpn/users/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		name, action := parts[0], parts[1]
		if !vpn.ValidName(name) {
			writeJSON(w, 400, map[string]string{"error": "bad name"})
			return
		}
		switch {
		case action == "qr" && r.Method == http.MethodGet:
			qp := vpn.QRPath(name)
			b, err := os.ReadFile(qp)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": "no qr"})
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(200)
			_, _ = w.Write(b)
		case action == "subscription" && r.Method == http.MethodGet:
			writeJSON(w, 410, map[string]string{"error": "subscription removed; use vless/core/hy2 links"})
			return
		case action == "subscription_legacy_disabled" && r.Method == http.MethodGet:
			sub, ok := vpn.ReadClient(name, "subscription.txt", "link.txt")
			if !ok {
				writeJSON(w, 404, map[string]string{"error": "not found"})
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(sub + "\n"))
		case action == "link" && r.Method == http.MethodGet:
			users, _ := vpn.ListNative()
			var uuid string
			for _, u := range users {
				if u.Name == name {
					uuid = u.UUID
					break
				}
			}
			vless := ""
			if uuid != "" {
				vless = vpn.PreferredVLESSLink(name, uuid)
				_ = vpn.WriteClientConfigs(name, uuid)
			}
			sub, _ := vpn.ReadClient(name, "subscription.txt", "link.txt")
			hy2, _ := vpn.ReadClient(name, "link-hy2.txt")
			core, _ := vpn.ReadClient(name, "link-vless-core.txt")
			writeJSON(w, 200, map[string]any{"name": name, "subscription": sub, "vless": vless, "hy2": hy2, "core": core})
		case action == "rename" && r.Method == http.MethodPost:
			newName, _ := readJSON(r)["new_name"].(string)
			newName = strings.TrimSpace(newName)
			out, err := vpn.Rename(name, newName)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			audit.Log("session", "vpn.rename", name, newName)
			writeJSON(w, 200, map[string]any{"ok": true, "name": out})
		case action == "note" && r.Method == http.MethodPost:
			note, _ := readJSON(r)["note"].(string)
			out, err := vpn.Note(name, note)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		case action == "disable" && r.Method == http.MethodPost:
			out, err := vpn.Disable(name)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		case action == "enable" && r.Method == http.MethodPost:
			out, err := vpn.Enable(name)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		case action == "revoke" && r.Method == http.MethodPost:
			out, err := vpn.Revoke(name)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": strings.TrimSpace(out)})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "output": strings.TrimSpace(out)})
		default:
			writeJSON(w, 404, map[string]string{"error": "not found"})
		}
	})

}
