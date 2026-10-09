package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/hardening"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func registerGitAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/git/repos", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			list, err := gitstore.ListInfo()
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"repos": list, "root": gitstore.Root()})
			return
		}
		if r.Method == http.MethodPost {
			body := readJSON(r)
			name, _ := body["name"].(string)
			path, err := gitstore.Init(name)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			audit.Log("session", "git.init", name, path)
			writeJSON(w, 200, map[string]any{
				"ok": true, "path": path,
				"remote_hint": gitstore.RemoteHint(name, hardening.SSHPort()),
			})
			return
		}
		if r.Method == http.MethodDelete {
			name := r.URL.Query().Get("name")
			if name == "" {
				body := readJSON(r)
				name, _ = body["name"].(string)
			}
			if err := gitstore.Delete(name); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			audit.Log("session", "git.delete", name, "")
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		writeJSON(w, 405, map[string]string{"error": "method"})
	})
	mux.HandleFunc("/api/git/log", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			name = r.URL.Query().Get("repo")
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		out, err := gitstore.Log(name, n)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error(), "log": out})
			return
		}
		writeJSON(w, 200, map[string]any{"log": out})
	})
	mux.HandleFunc("/api/git/show", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			name = r.URL.Query().Get("repo")
		}
		rev := r.URL.Query().Get("rev")
		if rev == "" {
			rev = r.URL.Query().Get("path")
		}
		out, err := gitstore.Show(name, rev)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error(), "show": out})
			return
		}
		writeJSON(w, 200, map[string]any{"show": out})
	})
	mux.HandleFunc("/api/git/pipelines", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		_ = gitstore.EnsureSamplePipeline()
		list, err := gitstore.ListPipelines()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"pipelines": list, "dir": gitstore.PipelineDir()})
	})
	mux.HandleFunc("/api/git/pipeline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		repo, _ := body["repo"].(string)
		pipe, _ := body["pipeline"].(string)
		out, err := gitstore.RunPipeline(repo, pipe)
		audit.Log("session", "git.pipeline", repo+"/"+pipe, "")
		if err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error(), "output": out})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "output": out})
	})

	mux.HandleFunc("/api/git/workflow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		repo, _ := body["repo"].(string)
		path, _ := body["path"].(string)
		out, err := gitstore.RunWorkflow(repo, path)
		audit.Log("session", "git.workflow", repo+"/"+path, "")
		if err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error(), "output": out})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "output": out})
	})

	mux.HandleFunc("/api/git/artifacts", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		repo := r.URL.Query().Get("repo")
		list, err := gitstore.ListArtifacts(repo)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"artifacts": list, "dir": gitstore.ArtifactDir()})
	})
	mux.HandleFunc("/api/git/artifact", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		path := r.URL.Query().Get("path")
		out, err := gitstore.ReadArtifact(path)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"path": path, "body": out})
	})

	mux.HandleFunc("/api/git/projects", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			list, err := gitstore.LoadProjects()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "projects": list})
		case http.MethodPost:
			var body gitstore.Project
			_ = json.NewDecoder(r.Body).Decode(&body)
			fetch := r.URL.Query().Get("fetch") != "0"
			if err := gitstore.AddProject(body, fetch); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "project": body})
		case http.MethodDelete:
			name := r.URL.Query().Get("name")
			if name == "" {
				writeJSON(w, 400, map[string]any{"ok": false, "error": "name required"})
				return
			}
			if err := gitstore.RemoveProject(name); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	
	mux.HandleFunc("/api/git/projects/detail", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !requireSession(w, r) {
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "name required"})
			return
		}
		d, err := gitstore.ProjectDetailOf(name)
		if err != nil {
			writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "project": d})
	})

	mux.HandleFunc("/api/git/projects/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		out, err := gitstore.SyncProject(body.Name)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error(), "log": out})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "log": out})
	})
	mux.HandleFunc("/api/git/projects/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var body struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		out, err := gitstore.BuildProject(body.Name, body.Ref)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error(), "log": out})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "log": out})
	})

	mux.HandleFunc("/api/git/projects/queue", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			pending := r.URL.Query().Get("pending") == "1"
			writeJSON(w, 200, map[string]any{"ok": true, "jobs": gitstore.ListMacQueue(pending)})
		case http.MethodPost:
			var body struct {
				Action string `json:"action"` // done|cancel
				ID     string `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			var err error
			switch body.Action {
			case "done", "complete":
				err = gitstore.CompleteMacBuild(body.ID)
			case "cancel":
				err = gitstore.CancelMacBuild(body.ID)
			default:
				writeJSON(w, 400, map[string]any{"ok": false, "error": "action done|cancel"})
				return
			}
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			writeJSON(w, 405, map[string]any{"ok": false, "error": "method"})
		}
	})

}
