package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/PavelNeyman/netductor/internal/aptguard"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func registerSessionAptGuard(mux *http.ServeMux) {
	mux.HandleFunc("/api/aptguard/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		state := paths.StateDir()
		host, _ := os.Hostname()
		if host == "" {
			host = "local"
		}
		rep, err := aptguard.LoadLatest(state, host)
		if err != nil {
			rep, err = aptguard.LoadLatest(state, "local")
		}
		out := map[string]any{
			"ok":     err == nil,
			"host":   host,
			"report": nil,
		}
		if err == nil {
			out["report"] = rep
			out["stale"] = aptguard.Stale(rep, 12*time.Hour)
		} else {
			out["error"] = "no report"
		}
		// secondary last_cmd hints if any
		list := secondary.List()
		var secs []map[string]string
		for _, d := range list {
			secs = append(secs, map[string]string{"id": d.ID, "last_cmd": d.LastCmd, "last_log": truncateStr(d.LastCmdLog, 200)})
		}
		out["secondaries"] = secs
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/api/aptguard/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		state := paths.StateDir()
		host, _ := os.Hostname()
		if host == "" {
			host = "local"
		}
		rep, _, err := aptguard.RunLocalScan(state, host, "/")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rep)
	})

	mux.HandleFunc("/api/aptguard/scan-secondary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.ID == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		if err := secondary.EnqueueCmd(body.ID, "aptguard_scan"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "queued": "aptguard_scan", "id": body.ID})
	})
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
