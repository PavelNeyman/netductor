package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/PavelNeyman/netductor/internal/audit"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/hardening"
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

	
	mux.HandleFunc("/api/git/build-host", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			c := gitstore.LoadBuildHost()
			writeJSON(w, 200, map[string]any{
				"ok": true,
				"config": c,
				"status": gitstore.BuildHostStatusLine(),
				"agents": gitstore.ListBuildHostStates(),
			})
		case http.MethodPost:
			var body gitstore.BuildHostConfig
			_ = json.NewDecoder(r.Body).Decode(&body)
			if err := gitstore.SaveBuildHost(body); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "status": gitstore.BuildHostStatusLine()})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/git/projects/ship-jobs", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			jobs, err := gitstore.ListShipJobs()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			state := r.URL.Query().Get("state")
			if state != "" {
				var f []gitstore.ShipJob
				for _, j := range jobs {
					if j.State == state {
						f = append(f, j)
					}
				}
				jobs = f
			}
			writeJSON(w, 200, map[string]any{"ok": true, "jobs": jobs, "vps_slot": gitstore.VpsSlotTaken(), "counts": gitstore.CountShipJobsByState()})
		case http.MethodPost:
			var body struct {
				Action string `json:"action"` // pause|resume|rm|complete
				ID     string `json:"id"`
				From   string `json:"from"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID == "" {
				writeJSON(w, 400, map[string]any{"ok": false, "error": "id required"})
				return
			}
			switch body.Action {
			case "pause":
				err := gitstore.RequestShipPause(body.ID)
				if err != nil {
					writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
					return
				}
				writeJSON(w, 200, map[string]any{"ok": true})
			case "resume":
				job, out, err := gitstore.ResumeShip(body.ID)
				resp := map[string]any{"ok": err == nil, "log": out, "job": job}
				if err != nil {
					resp["error"] = err.Error()
					writeJSON(w, 400, resp)
					return
				}
				writeJSON(w, 200, resp)
			case "rm", "delete":
				if err := gitstore.RemoveShipJob(body.ID); err != nil {
					writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
					return
				}
				writeJSON(w, 200, map[string]any{"ok": true})
			case "complete":
				job, out, err := gitstore.CompleteShip(body.ID, body.From)
				resp := map[string]any{"ok": err == nil, "log": out, "job": job}
				if err != nil {
					resp["error"] = err.Error()
					writeJSON(w, 400, resp)
					return
				}
				writeJSON(w, 200, resp)
			default:
				writeJSON(w, 400, map[string]any{"ok": false, "error": "action pause|resume|rm|complete"})
			}
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/git/projects/migrate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
			}
			return
		}
		var body struct {
			Owner  string `json:"owner"`
			DryRun bool   `json:"dry_run"`
			Sync   bool   `json:"sync"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Owner == "" {
			body.Owner = "PavelNeyman"
		}
		added, skipped, err := gitstore.MigrateFromGitHub(gitstore.MigrateOpts{
			Owner: body.Owner, DryRun: body.DryRun, Sync: body.Sync,
		})
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{
			"ok": true, "owner": body.Owner, "dry_run": body.DryRun, "sync": body.Sync,
			"added": added, "skipped": skipped,
			"added_n": len(added), "skipped_n": len(skipped),
		})
	})

	mux.HandleFunc("/api/git/projects/vps-queue", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			pending := r.URL.Query().Get("pending") == "1"
			writeJSON(w, 200, map[string]any{"ok": true, "jobs": gitstore.ListVpsQueue(pending)})
		case http.MethodPost:
			var body struct {
				Action string `json:"action"` // cancel
				ID     string `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Action == "cancel" && body.ID != "" {
				if err := gitstore.CancelVpsJob(body.ID); err != nil {
					writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
					return
				}
				writeJSON(w, 200, map[string]any{"ok": true})
				return
			}
			writeJSON(w, 400, map[string]any{"ok": false, "error": "action cancel + id required"})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
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
