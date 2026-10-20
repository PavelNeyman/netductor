package main

import (
	"encoding/json"
	"net/http"

	gitstore "github.com/PavelNeyman/netductor/internal/git"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func registerGitProjects(mux *http.ServeMux) {
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

	mux.HandleFunc("/api/git/projects/sync-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
			}
			return
		}
		results, okN, failN := gitstore.SyncAll()
		writeJSON(w, 200, map[string]any{"ok": failN == 0, "ok_n": okN, "fail_n": failN, "results": results})
	})
	mux.HandleFunc("/api/git/projects/build-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
			}
			return
		}
		queued, skipped, err := gitstore.ShipAllVPS()
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "queued": queued, "skipped": skipped, "queued_n": len(queued), "skipped_n": len(skipped)})
	})

	mux.HandleFunc("/api/git/projects/mirror-push", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
			}
			return
		}
		var body struct {
			Name string `json:"name"`
			All  bool   `json:"all"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.All || body.Name == "all" {
			list, err := gitstore.LoadProjects()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			var results []map[string]any
			for _, pr := range list {
				out, err := gitstore.MirrorPush(pr.Name)
				m := map[string]any{"name": pr.Name, "output": out, "ok": err == nil}
				if err != nil {
					m["error"] = err.Error()
				}
				results = append(results, m)
			}
			writeJSON(w, 200, map[string]any{"ok": true, "results": results})
			return
		}
		if body.Name == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "name or all required"})
			return
		}
		out, err := gitstore.MirrorPush(body.Name)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error(), "output": out})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "output": out})
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
		// hard cut: same as ship
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var body struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
			Tag  string `json:"tag"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Name == "netductor" {
			got, dst, logPath, err := ndupdate.Ship(ndupdate.ShipOpts{Tag: body.Tag})
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error(), "log": logPath})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "tag": got, "path": dst})
			return
		}
		tag := body.Tag
		if tag == "" {
			tag = gitstore.NextProjectTag(body.Name)
		}
		if r.URL.Query().Get("wait") == "1" {
			job, out, err := gitstore.ShipProject(gitstore.ShipOpts{Project: body.Name, Ref: body.Ref, Tag: tag})
			resp := map[string]any{"ok": err == nil, "log": out, "job": job}
			if err != nil {
				resp["error"] = err.Error()
				writeJSON(w, 400, resp)
				return
			}
			writeJSON(w, 200, resp)
			return
		}
		job, err := gitstore.EnqueueShipJob(gitstore.ShipOpts{Project: body.Name, Ref: body.Ref, Tag: tag})
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "async": true, "job": job})
	})

	mux.HandleFunc("/api/git/projects/ship", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var body struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
			Tag  string `json:"tag"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Name == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "name required"})
			return
		}
		if body.Name == "netductor" {
			got, dst, logPath, err := ndupdate.Ship(ndupdate.ShipOpts{Tag: body.Tag})
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error(), "log": logPath})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "tag": got, "path": dst})
			return
		}
		tag := body.Tag
		if tag == "" {
			tag = gitstore.NextProjectTag(body.Name)
		}
		if r.URL.Query().Get("wait") == "1" {
			job, out, err := gitstore.ShipProject(gitstore.ShipOpts{Project: body.Name, Ref: body.Ref, Tag: tag})
			resp := map[string]any{"ok": err == nil, "log": out, "job": job}
			if err != nil {
				resp["error"] = err.Error()
				writeJSON(w, 400, resp)
				return
			}
			writeJSON(w, 200, resp)
			return
		}
		job, err := gitstore.EnqueueShipJob(gitstore.ShipOpts{Project: body.Name, Ref: body.Ref, Tag: tag})
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "async": true, "job": job})
	})

}
