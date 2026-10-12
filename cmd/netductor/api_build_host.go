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
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		token = strings.TrimSpace(token)
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
		// redact tokens
		for i := range list {
			list[i].Token = ""
		}
		writeJSON(w, 200, map[string]any{
			"ok": true,
			"hosts": list,
			"config": gitstore.LoadBuildHost(),
			"status": gitstore.BuildHostStatusLine(),
		})
	})

	mux.HandleFunc("/api/build-host/deploy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var body struct {
			Host     string `json:"host"`
			User     string `json:"user"`
			Port     int    `json:"port"`
			Password string `json:"password"`
			KeyPath  string `json:"key_path"`
			Label    string `json:"label"`
			PrimaryURL string `json:"primary_url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		log, err := gitstore.DeployBuildHost(gitstore.DeployBuildHostOpts{
			Host: body.Host, User: body.User, Port: body.Port,
			Password: body.Password, KeyPath: body.KeyPath, Label: body.Label,
			PrimaryURL: body.PrimaryURL,
		})
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error(), "log": log})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "log": log})
	})
}
