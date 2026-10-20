package main

import (
	"encoding/json"
	"net/http"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func registerGitShip(mux *http.ServeMux) {
	mux.HandleFunc("/api/git/build-host", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			c := gitstore.LoadBuildHost()
			writeJSON(w, 200, map[string]any{
				"ok":     true,
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
