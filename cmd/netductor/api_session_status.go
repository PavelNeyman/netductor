package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/probes"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/session"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerSessionStatus(mux *http.ServeMux) {
	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		exp, ok := session.Expiry(tok)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "expires_unix": exp})
	})
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, metrics.Collect())
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, metrics.Collect())
	})
	mux.HandleFunc("/metrics/prometheus", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		text := metrics.PrometheusText()
		if st := primaryJobStore(); st != nil {
			run, fail := st.Counts()
			text += "\n# HELP netductor_jobs_running Jobs currently running or queued\n# TYPE netductor_jobs_running gauge\n"
			text += fmt.Sprintf("netductor_jobs_running %d\n", run)
			text += "# HELP netductor_jobs_failed_total Failed jobs in store\n# TYPE netductor_jobs_failed_total gauge\n"
			text += fmt.Sprintf("netductor_jobs_failed_total %d\n", fail)
		}
		_, _ = w.Write([]byte(text))
	})
	mux.HandleFunc("/api/metrics/history", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		limit := 180
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				limit = n
			}
		}
		writeJSON(w, 200, map[string]any{"points": metrics.History(limit)})
	})

	mux.HandleFunc("/api/probes", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"probes": probes.Run(probes.Load()), "config": probes.Load()})
	})
	mux.HandleFunc("/api/latest", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"metrics": metrics.Collect(), "probes": probes.Run(probes.Load())})
	})
	mux.HandleFunc("/api/probes/uptime", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		limit := 1440
		if s := r.URL.Query().Get("limit"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				limit = n
			}
		}
		writeJSON(w, 200, map[string]any{"uptime": metrics.ProbeUptime(limit)})
	})
	mux.HandleFunc("/api/probes/config", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, probes.Load())
		case http.MethodPost, http.MethodPut:
			cfg := readJSON(r)
			if err := probes.Save(cfg); err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/doctor", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		// Capture CLI text while building structured report
		rep := CollectDoctor()
		writeJSON(w, 200, rep)
	})
	mux.HandleFunc("/api/domain", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		keys := []string{"NETDUCTOR_DOMAIN", "NETDUCTOR_PUBLIC_HOSTNAME", "NETDUCTOR_CORE_HOST", "NETDUCTOR_VPN_HOST", "NETDUCTOR_REDIRECT_BASE"}
		m := map[string]string{}
		for _, k := range keys {
			m[k] = os.Getenv(k)
		}
		for _, f := range []string{"public_hostname", "vpn_hostname"} {
			if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), f)); err == nil {
				m["file:"+f] = strings.TrimSpace(string(b))
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true, "domain": m})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		m := metrics.Collect()
		mm := vpn.CollectMismatch(30)
		relays := []map[string]any{}
		for _, d := range secondary.List() {
			relays = append(relays, map[string]any{
				"id": d.ID, "name": d.Name, "ip": d.PublicIP,
				"mismatch_total": d.MismatchTotal, "mismatch_by_ip": d.MismatchByIP,
			})
		}
		writeJSON(w, 200, map[string]any{
			"ok": true, "service": "netductor", "version": ndver.Release,
			"metrics": m, "probes": probes.Run(probes.Load()),
			"mismatch": mm, "secondary_mismatch": relays,
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		m := metrics.Collect()
		mm := vpn.CollectMismatch(30)
		writeJSON(w, 200, map[string]any{"ok": true, "metrics": m, "probes": probes.Run(probes.Load()), "mismatch": mm})
	})

	mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"events": audit.Tail(100)})
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]any{"sessions": session.List()})
			return
		}
		if r.Method == http.MethodDelete || (r.Method == http.MethodPost && r.URL.Query().Get("action") == "revoke-all") {
			_ = session.RevokeAll()
			audit.Log("session", "session.revoke_all", "", "")
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
	})

}
