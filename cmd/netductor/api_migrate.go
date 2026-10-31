package main

import (
	"context"
	"net/http"
	"os"

	"github.com/PavelNeyman/netductor/internal/migrate"
)

func registerMigrateAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/migrate/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
			return
		}
		rec, err := migrate.LoadJournalPublic()
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "records": rec})
	})
	mux.HandleFunc("/api/migrate/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
			return
		}
		if os.Getenv("NETDUCTOR_SKIP_MIGRATE") == "1" {
			writeJSON(w, 503, map[string]any{"ok": false, "error": "NETDUCTOR_SKIP_MIGRATE=1"})
			return
		}
		res, err := migrate.RunAll(context.Background())
		if err != nil {
			writeJSON(w, 500, map[string]any{
				"ok": false, "error": err.Error(),
				"ran": res.Ran, "skipped": res.Skipped, "failed": res.Failed,
			})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "ran": res.Ran, "skipped": res.Skipped})
	})
	mux.HandleFunc("/api/migrate/clear-failed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if !requireSession(w, r) {
			return
		}
		if err := migrate.ClearFailedRecords(); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
}
