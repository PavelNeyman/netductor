package main

import (
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
		writeJSON(w, 405, map[string]string{"error": "method"})
	})
	mux.HandleFunc("/api/git/log", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		name := r.URL.Query().Get("name")
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
		rev := r.URL.Query().Get("rev")
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
}
