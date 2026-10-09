package main

import (
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerPolicyAPI(mux *http.ServeMux) {
	registerPolicyPresetAPI(mux)
	mux.HandleFunc("/api/services", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			c, err := policy.EnsureCatalog()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "catalog": c})
		case http.MethodPost:
			body := readJSON(r)
			c, err := policy.EnsureCatalog()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			s := policy.Service{
				ID:          strBody(body, "id"),
				Title:       strBody(body, "title"),
				Kind:        strBody(body, "kind"),
				Description: strBody(body, "description"),
			}
			if s.Kind == "" {
				s.Kind = policy.KindInternal
			}
			if v, ok := body["disabled"].(bool); ok {
				s.Disabled = v
			}
			if err := c.Upsert(s); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			if err := policy.SaveCatalog(c); err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			audit.Log("session", "services.upsert", s.ID, s.Kind)
			writeJSON(w, 200, map[string]any{"ok": true, "catalog": c})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/services/", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/services/")
		id = strings.Trim(id, "/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodDelete {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		c, err := policy.EnsureCatalog()
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if err := c.SoftDisable(id); err != nil {
			writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if err := policy.SaveCatalog(c); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		audit.Log("session", "services.disable", id, "")
		writeJSON(w, 200, map[string]any{"ok": true, "catalog": c})
	})

	// Edge device policy (Users policy is under /vpn/users/{name}/policy)
	mux.HandleFunc("/api/edge/device-policy", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			body := readJSON(r)
			id = strBody(body, "id")
			if id == "" {
				id = strBody(body, "device_id")
			}
		}
		if id == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "id required"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			p, err := edge.GetDevicePolicy(id)
			if err != nil {
				writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "subject_type": "edge", "subject_id": id, "policy": p})
		case http.MethodPut, http.MethodPost:
			body := readJSON(r)
			if id == "" {
				id = strBody(body, "id")
			}
			p, err := policyFromBody(body)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			if err := edge.SetDevicePolicy(id, p, "api"); err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			out, _ := edge.GetDevicePolicy(id)
			st, _ := policy.ApplyRoutes()
			audit.Log("session", "edge.policy", id, "")
			writeJSON(w, 200, map[string]any{"ok": true, "policy": out, "apply": st})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/policy/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		st, err := policy.ApplyRoutes()
		if err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error(), "apply": st})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "apply": st})
	})
}

func strBody(body map[string]any, k string) string {
	v, _ := body[k].(string)
	return strings.TrimSpace(v)
}

func policyFromBody(body map[string]any) (policy.AccessPolicy, error) {
	p := policy.DefaultUserPolicy()
	if v, ok := body["allow_internet"].(bool); ok {
		p.AllowInternet = v
	}
	if v, ok := body["services_mode"].(string); ok {
		p.ServicesMode = strings.TrimSpace(v)
	}
	switch s := body["services"].(type) {
	case []any:
		for _, x := range s {
			if str, ok := x.(string); ok {
				p.Services = append(p.Services, str)
			}
		}
	case []string:
		p.Services = append(p.Services, s...)
	}
	p.Normalize()
	return p, nil
}

func registerPolicyPresetAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/policy/presets", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			list, err := policy.LoadCustomPresets()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{
				"ok": true,
				"builtin": []string{policy.PresetFull, policy.PresetMedia, policy.PresetNone},
				"custom": list,
			})
		case http.MethodPost:
			body := readJSON(r)
			p := policy.CustomPreset{
				ID:    strBody(body, "id"),
				Title: strBody(body, "title"),
			}
			if v, ok := body["allow_internet"].(bool); ok {
				p.AllowInternet = v
			} else {
				p.AllowInternet = true
			}
			p.ServicesMode = strBody(body, "services_mode")
			if arr, ok := body["services"].([]any); ok {
				for _, x := range arr {
					if s, ok := x.(string); ok && s != "" {
						p.Services = append(p.Services, s)
					}
				}
			}
			out, err := policy.UpsertCustomPreset(p)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			audit.Log("session", "policy.preset.upsert", out.ID, out.Title)
			writeJSON(w, 200, map[string]any{"ok": true, "preset": out})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/policy/presets/", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/policy/presets/")
		id = strings.Trim(id, "/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			if err := policy.DeleteCustomPreset(id); err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			audit.Log("session", "policy.preset.delete", id, "")
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/api/policy/presets/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		user := strBody(body, "user")
		name := strBody(body, "preset")
		if user == "" || name == "" {
			writeJSON(w, 400, map[string]any{"ok": false, "error": "user and preset required"})
			return
		}
		cur, _ := vpn.GetUserPolicy(user)
		next := policy.ApplyNamedPreset(name, cur)
		if err := vpn.SetUserPolicy(user, next, "api"); err != nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		audit.Log("session", "policy.preset.apply", user, name)
		writeJSON(w, 200, map[string]any{"ok": true, "user": user, "policy": next})
	})
}
