package main

import (
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
)

func registerEdgeTemplates(mux *http.ServeMux) {
	mux.HandleFunc("/api/edge/template", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		did := r.URL.Query().Get("device_id")
		if did == "" {
			did = edge.DeviceIDFromAuth(auth)
		}
		if r.Method == http.MethodGet {
			if !edge.RequireApproved(auth, did) {
				writeJSON(w, 403, map[string]string{"error": "not_approved"})
				return
			}
			tmpl, err := edge.TemplateWithVPN(did)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, tmpl)
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
	})
	mux.HandleFunc("/api/edge/templates", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		edge.EnsureDefaultTemplate()
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{"templates": edge.ListTemplates()})
		case http.MethodPost:
			body := readJSON(r)
			id, _ := body["id"].(string)
			if id == "" {
				writeJSON(w, 400, map[string]string{"error": "id"})
				return
			}
			// Default: merge into existing template (avoid wiping wifi/guest when body is partial).
			// Full replace only with replace=true|1|yes.
			replace := false
			switch v := body["replace"].(type) {
			case bool:
				replace = v
			case string:
				replace = v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
			}
			delete(body, "replace")
			tmpl, err := edge.MergeTemplate(id, edge.Template(body), replace)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "id": id, "replace": replace, "template": tmpl})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/edge/templates/vpn", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		edge.EnsureDefaultTemplate()
		switch r.Method {
		case http.MethodGet:
			id := r.URL.Query().Get("id")
			if id == "" {
				id = "default"
			}
			tmpl, err := edge.GetTemplate(id)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			vpn, _ := tmpl["vpn"].(map[string]any)
			if vpn == nil {
				vpn = map[string]any{}
			}
			writeJSON(w, 200, map[string]any{"id": id, "vpn": vpn})
		case http.MethodPost:
			body := readJSON(r)
			id, _ := body["id"].(string)
			if id == "" {
				id = "default"
			}
			// accept nested vpn{} or flat keys
			kvs := map[string]any{}
			if nested, ok := body["vpn"].(map[string]any); ok {
				for k, v := range nested {
					kvs[k] = v
				}
			}
			for _, k := range []string{"enabled", "mode", "dns", "dns_mode", "fallback", "soft_fallback"} {
				if v, ok := body[k]; ok {
					kvs[k] = v
				}
			}
			tmpl, err := edge.SetTemplateVPN(id, kvs)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "id": id, "vpn": tmpl["vpn"]})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/edge/bind-template", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		tid, _ := body["template_id"].(string)
		ov, _ := body["overlay"].(map[string]any)
		if err := edge.BindTemplate(did, tid, ov); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
}
