package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/stack"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

func registerUpdateAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/update/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		force := r.URL.Query().Get("force") == "1" || r.URL.Query().Get("refresh") == "1"
		if force {
			writeJSON(w, 200, ndupdate.CheckStatusForce(ndver.Release))
			return
		}
		writeJSON(w, 200, ndupdate.CheckStatus(ndver.Release))
	})
	mux.HandleFunc("/api/update/github-token", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, ndupdate.GetTokensStatus())
		case http.MethodPost:
			var body struct {
				Token string `json:"token"`
				Clear bool   `json:"clear"`
				Kind  string `json:"kind"` // releases | repos
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			kind := body.Kind
			if kind == "" {
				kind = ndupdate.TokenKindReleases
			}
			if body.Clear {
				_ = ndupdate.ClearTokenKind(kind)
				writeJSON(w, 200, map[string]any{"ok": true, "status": ndupdate.GetTokensStatus()})
				return
			}
			tok := strings.TrimSpace(body.Token)
			if tok == "" {
				writeJSON(w, 400, map[string]string{"error": "token required (or clear:true)"})
				return
			}
			if err := ndupdate.SetTokenKind(kind, tok); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "status": ndupdate.GetTokensStatus()})
		case http.MethodDelete:
			kind := r.URL.Query().Get("kind")
			if kind == "" {
				kind = ndupdate.TokenKindReleases
			}
			_ = ndupdate.ClearTokenKind(kind)
			writeJSON(w, 200, map[string]any{"ok": true, "status": ndupdate.GetTokensStatus()})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/update/releases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		limit := 15
		if v := r.URL.Query().Get("limit"); v != "" {
			fmt.Sscanf(v, "%d", &limit)
		}
		force := r.URL.Query().Get("force") == "1" || r.URL.Query().Get("refresh") == "1"
		var list []ndupdate.ReleaseInfo
		var err error
		if force {
			list, err = ndupdate.ListReleasesForce(limit)
		} else {
			list, err = ndupdate.ListReleases(limit)
		}
		if err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "local": ndver.Release, "releases": list, "forced": force})
	})
	mux.HandleFunc("/api/update/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
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
		// node: single path — stack apply does one backup; no double-wait here.
		if comp == "node" {
			if err := stack.ScheduleApply(tag); err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "scheduled": true, "version": tag})
			return
		}
		backupPath := ""
		if !body.NoBackup {
			if p, err := install.Backup(); err != nil {
				backupPath = "error:" + err.Error()
			} else {
				backupPath = p
				acked, pend := install.WaitForBackupPull(20 * time.Second)
				backupPath = fmt.Sprintf("%s (pull acked=%d pending=%d)", p, acked, pend)
			}
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
