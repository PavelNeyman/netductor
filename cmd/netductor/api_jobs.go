package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/jobs"
)

var (
	primaryJobsOnce sync.Once
	primaryJobs     *jobs.Store
)

func primaryJobStore() *jobs.Store {
	primaryJobsOnce.Do(func() {
		dir := filepath.Join("/var/lib/netductor", "jobs")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			dir = filepath.Join(os.TempDir(), "netductor-jobs")
			_ = os.MkdirAll(dir, 0o700)
		}
		primaryJobs = jobs.NewStore(dir)
		primaryJobs.Register("stack.apply", runJobStackApply)
		primaryJobs.Register("doctor.run", runJobDoctor)
	})
	return primaryJobs
}

func registerJobsAPI(mux *http.ServeMux) {
	_ = primaryJobStore()
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			st := jobs.Status(r.URL.Query().Get("status"))
			typ := r.URL.Query().Get("type")
			writeJSON(w, 200, map[string]any{"jobs": primaryJobStore().List(st, typ)})
		case http.MethodPost:
			body := readJSON(r)
			typ, _ := body["type"].(string)
			params, _ := body["params"].(map[string]any)
			if params == nil {
				params = map[string]any{}
			}
			if typ == "" {
				writeJSON(w, 400, map[string]string{"error": "type required"})
				return
			}
			j, err := primaryJobStore().Start(typ, params)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 202, j.Snapshot())
		default:
			writeJSON(w, 405, map[string]string{"error": "GET or POST"})
		}
	})
	mux.HandleFunc("/api/jobs/", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		id := parts[0]
		if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			if err := primaryJobStore().Cancel(id); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "id": id})
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		j := primaryJobStore().Get(id)
		if j == nil {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, 200, j.Snapshot())
	})
}

func runJobStackApply(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	ver := ""
	if j.Params != nil {
		if v, ok := j.Params["version"].(string); ok {
			ver = strings.TrimSpace(v)
		}
	}
	setStep(jobs.Step{ID: "stack", Message: "stack apply " + ver})
	args := []string{"stack", "apply"}
	if ver != "" {
		args = append(args, ver)
	}
	cmd := exec.CommandContext(ctx, "netductor", args...)
	out, err := cmd.CombinedOutput()
	appendLog(string(out))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(out))
	}
	setStep(jobs.Step{ID: "stack", Message: "ok", Done: true})
	return map[string]any{"ok": true, "log": string(out)}, nil
}

func runJobDoctor(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	setStep(jobs.Step{ID: "doctor", Message: "running doctor"})
	cmd := exec.CommandContext(ctx, "netductor", "doctor")
	out, err := cmd.CombinedOutput()
	appendLog(string(out))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(out))
	}
	setStep(jobs.Step{ID: "doctor", Message: "ok", Done: true})
	return map[string]any{"ok": true, "report": string(out)}, nil
}

