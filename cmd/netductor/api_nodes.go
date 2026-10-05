package main

import (
	"net/http"
	"strconv"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/session"
)

func registerNodesAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		list, err := nodes.List()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"nodes": list})
	})
	mux.HandleFunc("/api/nodes/self", func(w http.ResponseWriter, r *http.Request) {
		// loopback (local install scripts) OR operator session
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST"})
			return
		}
		if !isLoopback(r) && !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		hn, _ := body["hostname"].(string)
		role, _ := body["role"].(string)
		ip, _ := body["public_ip"].(string)
		if role == "" {
			role = "core"
		}
		if err := nodes.SelfRegisterLocal(hn, role, ip); err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/nodes/hostname", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST"})
			return
		}
		body := readJSON(r)
		id, _ := body["id"].(string)
		hn, _ := body["hostname"].(string)
		if id == "" || hn == "" {
			writeJSON(w, 400, map[string]any{"error": "id and hostname required"})
			return
		}
		n, err := nodes.SetDesiredHostname(id, hn)
		if err == nil {
			audit.Log("session", "nodes.rename", id, hn)
		}
		if err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "node": n})
	})

	// R7b: issue session from loopback only (op SSH tunnel / local curl) — no hours in shell string on node.
	mux.HandleFunc("/api/session/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		if !isLoopback(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "loopback_only"})
			return
		}
		body := readJSON(r)
		hours := 8
		if v, ok := body["hours"].(float64); ok {
			hours = int(v)
		}
		if v, ok := body["hours"].(string); ok {
			if n, err := strconv.Atoi(v); err == nil {
				hours = n
			}
		}
		label, _ := body["label"].(string)
		if label == "" {
			label = "op-issue"
		}
		tok, exp, err := session.Create(hours, label, clientIP(r))
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "session.issue", label, "")
		writeJSON(w, 200, map[string]any{"ok": true, "token": tok, "expires_unix": exp, "hours": hours})
	})

	mux.HandleFunc("/api/session/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		tok := bearer(r)
		if !session.Valid(tok) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		body := readJSON(r)
		if all, _ := body["all"].(bool); all {
			_ = session.RevokeAll()
		} else {
			session.Revoke(tok)
		}
		audit.Log("session", "session.revoke", "", "")
		writeJSON(w, 200, map[string]bool{"ok": true})
	})
}
