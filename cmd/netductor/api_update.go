package main

import (
	"fmt"
	"time"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/install"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

func registerUpdateAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/update/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, ndupdate.CheckStatus(ndver.Release))
	})
	mux.HandleFunc("/api/update/releases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		limit := 15
		list, err := ndupdate.ListReleases(limit)
		if err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "local": ndver.Release, "releases": list})
	})
	mux.HandleFunc("/api/update/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Version   string `json:"version"`
			Component string `json:"component"`
			NoBackup  bool   `json:"no_backup"`
			NoRestart bool   `json:"no_restart"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		comp := strings.TrimSpace(body.Component)
		if comp == "" {
			comp = "node"
		}
		tag := strings.TrimSpace(body.Version)
		if tag == "" {
			var err error
			tag, err = ndupdate.LatestReleaseTag()
			if err != nil {
				writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
				return
			}
		}
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag
		}
		backupPath := ""
		if !body.NoBackup {
			if p, err := install.Backup(); err != nil {
				backupPath = "error:" + err.Error()
			} else {
				backupPath = p
				acked, pend := install.WaitForBackupPull(45 * time.Second)
				backupPath = fmt.Sprintf("%s (pull acked=%d pending=%d)", p, acked, pend)
			}
		}
		dest := "/usr/local/bin/netductor"
		unit := "netductor-api"
		switch comp {
		case "tg", "telegram":
			dest = "/usr/local/bin/netductor-tg"
			unit = "netductor-telegram-bot"
		case "agent":
			dest = "/usr/local/bin/netductor-agent"
			unit = "netductor-secondary-agent"
		case "node", "netductor", "":
			comp = "node"
		default:
			writeJSON(w, 400, map[string]any{"ok": false, "error": "unknown component"})
			return
		}
		if err := ndupdate.DownloadReleaseAsset(tag, comp, dest); err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error(), "backup": backupPath})
			return
		}
		ndupdate.WriteVERSION(tag)
		if !body.NoRestart && unit != "" {
			_ = exec.Command("systemctl", "try-restart", unit).Run()
			if unit == "netductor-api" {
				_ = exec.Command("systemctl", "try-restart", "netductor-telegram-bot").Run()
				_ = exec.Command("systemctl", "try-restart", "netductor-redirect").Run()
			}
		}
		writeJSON(w, 200, map[string]any{
			"ok": true, "component": comp, "version": tag, "dest": dest, "backup": backupPath,
		})
	})
}
