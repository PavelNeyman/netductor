package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/PavelNeyman/netductor/internal/channels"
)

func registerChannelAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/channel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, channels.Collect())
	})
	mux.HandleFunc("/api/channel/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		mins := 60
		if v := r.URL.Query().Get("minutes"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 24*60 {
				mins = n
			}
		}
		id := r.URL.Query().Get("id")
		since := time.Now().UTC().Add(-time.Duration(mins) * time.Minute)
		pts, err := channels.ReadHistory(since, id)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		if pts == nil {
			pts = []channels.SamplePoint{}
		}
		writeJSON(w, 200, map[string]any{
			"minutes": mins,
			"id":      id,
			"points":  pts,
		})
	})
	mux.HandleFunc("/api/channel/sample", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !requireSession(w, r) {
			return
		}
		pt, err := channels.SampleOnce()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, pt)
	})
}
