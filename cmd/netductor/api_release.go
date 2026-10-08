package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func registerReleaseAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/release/local", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		tags, err := ndupdate.ListLocalTags()
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{
			"ok":   true,
			"root": ndupdate.LocalReleasesDir(),
			"tags": tags,
		})
	})

	mux.HandleFunc("/api/release/local/detail", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		tag := strings.TrimSpace(r.URL.Query().Get("tag"))
		if tag == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "tag required"})
			return
		}
		dir := ndupdate.LocalTagDir(tag)
		ents, err := os.ReadDir(dir)
		if err != nil {
			writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error(), "dir": dir})
			return
		}
		var files []map[string]any
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			fi, _ := e.Info()
			sz := int64(0)
			if fi != nil {
				sz = fi.Size()
			}
			files = append(files, map[string]any{"name": e.Name(), "size": sz})
		}
		writeJSON(w, 200, map[string]any{"ok": true, "tag": tag, "dir": dir, "files": files})
	})

	mux.HandleFunc("/api/release/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
			return
		}
		var body struct {
			Tag        string `json:"tag"`
			SkipDarwin bool   `json:"skip_darwin"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.TrimSpace(body.Tag) == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "tag required"})
			return
		}
		if err := ndupdate.ScheduleBuild(body.Tag, body.SkipDarwin); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "scheduled": true, "tag": body.Tag, "log": "/var/log/netductor-release-build.log"})
	})

	mux.HandleFunc("/api/release/mirror-fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
			return
		}
		name := "netductor"
		var body struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.TrimSpace(body.Name) != "" {
			name = body.Name
		}
		if _, err := gitstore.MirrorEnsure(name, body.URL); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		out, err := gitstore.MirrorFetch(name)
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error(), "log": out})
			return
		}
		tags, _ := gitstore.ListTags(name)
		writeJSON(w, 200, map[string]any{"ok": true, "name": name, "log": out, "tags": tags})
	})

	mux.HandleFunc("/api/release/git-tags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "netductor"
		}
		tags, err := gitstore.ListTags(name)
		if err != nil {
			writeJSON(w, 200, map[string]any{"ok": true, "tags": []string{}, "error": err.Error(), "hint": "mirror-fetch first"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "name": name, "tags": tags})
	})

	mux.HandleFunc("/api/release/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
			return
		}
		var body struct {
			Tag string `json:"tag"`
			Dir string `json:"dir"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		dir := filepath.Clean(strings.TrimSpace(body.Dir))
		if body.Tag == "" || dir == "" || dir == "." || strings.Contains(dir, "..") {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "tag and absolute dir required"})
			return
		}
		if !filepath.IsAbs(dir) {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "dir must be absolute"})
			return
		}
		dst, err := ndupdate.ImportReleaseDir(body.Tag, dir)
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "dir": dst})
	})
}
