package main

import (
	"net/http"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/mtls"
)

func registerSessionMTLS(mux *http.ServeMux) {
	mux.HandleFunc("/api/mtls/certs", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		clients, _ := mtls.ListClientCerts()
		writeJSON(w, 200, map[string]any{
			"plane":   mtls.ListPlaneCerts(),
			"clients": clients,
			"revoked": func() any { l, _ := mtls.ListRevoked(); return l }(),
		})
	})
	mux.HandleFunc("/api/mtls/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		node, _ := body["node_id"].(string)
		serial, _ := body["serial"].(string)
		reason, _ := body["reason"].(string)
		if reason == "" {
			reason = "api"
		}
		if node != "" {
			ser, err := mtls.RevokeNodeCert(node, reason)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			audit.Log("session", "mtls.revoke", node, ser)
			writeJSON(w, 200, map[string]any{"ok": true, "serial": ser})
			return
		}
		if serial == "" {
			writeJSON(w, 400, map[string]string{"error": "node_id or serial required"})
			return
		}
		if err := mtls.RevokeSerial(serial, "", reason); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "mtls.revoke", serial, reason)
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/mtls/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		node, _ := body["node_id"].(string)
		if node == "" {
			writeJSON(w, 400, map[string]string{"error": "node_id required"})
			return
		}
		info, err := mtls.RotateClientFor(node, "api-rotate")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		audit.Log("session", "mtls.rotate", node, info.Serial)
		writeJSON(w, 200, map[string]any{"ok": true, "cert": info, "path": mtls.ClientDir(node)})
	})
}
