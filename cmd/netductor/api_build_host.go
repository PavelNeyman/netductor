package main

import (
	"encoding/json"
	"net/http"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func registerBuildHostAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/build-host/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "token"})
			return
		}
		var facts map[string]any
		_ = json.NewDecoder(r.Body).Decode(&facts)
		if err := gitstore.RecordBuildHostHeartbeat(token, facts); err != nil {
			writeJSON(w, 401, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/build-host/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		list := gitstore.ListBuildHostStates()
		for i := range list {
			list[i].Token = ""
		}
		writeJSON(w, 200, map[string]any{"ok": true, "hosts": list, "config": gitstore.LoadBuildHost(), "status": gitstore.BuildHostStatusLine()})
	})
		mux.HandleFunc("/api/build-host/jobs/claim", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "token"})
			return
		}
		hostID := ""
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body != nil {
			if v, ok := body["device_id"].(string); ok {
				hostID = v
			}
		}
		// validate token belongs to a build host
		okTok := false
		for _, s := range gitstore.ListBuildHostStates() {
			full, err := gitstore.LoadBuildHostState(s.ID)
			if err == nil && full.Token == token {
				okTok = true
				if hostID == "" {
					hostID = full.ID
				}
				break
			}
		}
		if !okTok {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "token"})
			return
		}
		job, tarPath, err := gitstore.ClaimAgentBuildJob(hostID)
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if job == nil {
			writeJSON(w, 200, map[string]any{"ok": true, "job": nil})
			return
		}
		writeJSON(w, 200, map[string]any{
			"ok": true,
			"job": map[string]any{
				"id": job.ID, "project": job.Project, "ref": job.Ref, "tag": job.Tag,
				"work_url": "/api/build-host/jobs/" + job.ID + "/work.tgz",
			},
			"tar_path": tarPath,
		})
	})
	mux.HandleFunc("/api/build-host/jobs/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/build-host/jobs/")
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if strings.HasSuffix(path, "/work.tgz") && r.Method == http.MethodGet {
			id := strings.TrimSuffix(path, "/work.tgz")
			id = strings.TrimSuffix(id, "/")
			p := gitstore.ShipJobArtifactsDir(id) + "/work.tgz"
			http.ServeFile(w, r, p)
			return
		}
		if strings.HasSuffix(path, "/result") && r.Method == http.MethodPost {
			id := strings.TrimSuffix(path, "/result")
			id = strings.Trim(id, "/")
			var body struct {
				OK    bool   `json:"ok"`
				Log   string `json:"log"`
				Error string `json:"error"`
				Host  string `json:"host_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = token
			if err := gitstore.CompleteAgentBuild(id, body.Host, body.OK, body.Log, body.Error); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/build-host/deploy", func(w http.ResponseWriter, r *http.Request) {
		// Deploy is operator-only (Mac). Node must not orchestrate build-host.
		writeJSON(w, 410, map[string]any{
			"ok": false,
			"error": "build-host deploy only from operator (netductor-op tui / deploy build-host)",
		})
	})
}
