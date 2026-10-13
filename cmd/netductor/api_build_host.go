package main

import (
	"encoding/json"
	"net/http"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func registerBuildHostAgentAPI(mux *http.ServeMux) {
	// Agent plane (:8789): same handlers; token auth inside agent routes.
	registerBuildHostAPI(mux)
}

func registerBuildHostAPI(mux *http.ServeMux) {

	mux.HandleFunc("/api/build-host/backup-mirror", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		if err := gitstore.QueueBuildHostBackupMirror(); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		n, _ := gitstore.EnqueueBackupPullAllBuildHosts()
		writeJSON(w, 200, map[string]any{"ok": true, "queued_pull": n})
	})
	mux.HandleFunc("/api/build-host/recovery-arm", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		n := 0
		for _, s := range gitstore.ListBuildHostStates() {
			if s.Online {
				if gitstore.EnqueueBuildHostCmd(s.ID, "recovery_arm") == nil {
					n++
				}
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true, "queued": n})
	})

	mux.HandleFunc("/api/build-host/backup/latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		okTok := false
		for _, s := range gitstore.ListBuildHostStates() {
			if full, e := gitstore.LoadBuildHostState(s.ID); e == nil && full.Token == token {
				okTok = true
				break
			}
		}
		if !okTok {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "token"})
			return
		}
		_ = gitstore.QueueBuildHostBackupMirror()
		name, path, err := gitstore.LatestBuildBackupMirror()
		if err != nil {
			writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		w.Header().Set("X-Backup-Name", name)
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, path)
	})

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
		hostID := ""
		if facts != nil {
			if v, ok := facts["device_id"].(string); ok {
				hostID = v
			}
		}
		if hostID == "" {
			for _, s := range gitstore.ListBuildHostStates() {
				if full, e := gitstore.LoadBuildHostState(s.ID); e == nil && full.Token == token {
					hostID = full.ID
					break
				}
			}
		}
		cmds := gitstore.TakeBuildHostCmds(hostID)
		writeJSON(w, 200, map[string]any{"ok": true, "cmds": cmds})
	})
	mux.HandleFunc("/api/build-host/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		list := gitstore.ListBuildHostStates()
		for i := range list {
			list[i].Token = ""
		}
		waiting := 0
		if jobs, err := gitstore.ListShipJobs(); err == nil {
			for _, j := range jobs {
				if j.State == gitstore.ShipWaitingExternal && j.Reason == gitstore.ReasonBuildAgent {
					waiting++
				}
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true, "hosts": list, "config": gitstore.LoadBuildHost(), "status": gitstore.BuildHostStatusLine(), "waiting_build": waiting})
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
		// Bind host strictly to the token. Ignore client device_id.
		hostID, okTok := gitstore.AuthBuildHostToken(token)
		if !okTok || hostID == "" {
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
		hostID, ok := authBuildHost(r)
		if !ok {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "token"})
			return
		}
		if strings.HasSuffix(path, "/work.tgz") && r.Method == http.MethodGet {
			id := strings.TrimSuffix(path, "/work.tgz")
			id = strings.TrimSuffix(id, "/")
			if !jobOwnedBy(id, hostID) {
				writeJSON(w, 403, map[string]any{"ok": false, "error": "not claimed by this host"})
				return
			}
			p := gitstore.ShipJobArtifactsDir(id) + "/work.tgz"
			http.ServeFile(w, r, p)
			return
		}
		if strings.HasSuffix(path, "/artifacts") && r.Method == http.MethodPost {
			id := strings.TrimSuffix(path, "/artifacts")
			id = strings.Trim(id, "/")
			if !jobOwnedBy(id, hostID) {
				writeJSON(w, 403, map[string]any{"ok": false, "error": "not claimed by this host"})
				return
			}
			if err := gitstore.StoreAgentArtifacts(id, r.Body); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
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
			if body.Host != "" && body.Host != hostID {
				writeJSON(w, 403, map[string]any{"ok": false, "error": "host mismatch"})
				return
			}
			if err := gitstore.CompleteAgentBuild(id, hostID, body.OK, body.Log, body.Error); err != nil {
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

func authBuildHost(r *http.Request) (string, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return gitstore.AuthBuildHostToken(token)
}

func jobOwnedBy(jobID, hostID string) bool {
	m := gitstore.LoadAgentMeta(jobID)
	return m.HostID != "" && m.HostID == hostID
}
