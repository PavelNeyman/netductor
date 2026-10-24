package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/inventory"
	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func registerSecondariesAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/secondaries", handleSecondaries)
	mux.HandleFunc("/api/secondaries/", handleSecondaryByID)
	mux.HandleFunc("/api/svc-paths/peers", handleSvcPeers)
}

func handleSecondaries(w http.ResponseWriter, r *http.Request) {
	path := inventory.DefaultFile
	switch r.Method {
	case http.MethodGet:
		items, err := inventory.List(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSONStatus(w, http.StatusOK, items)
	case http.MethodPost:
		var body struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			PublicIP string `json:"public_ip"`
			Notes    string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		s, err := inventory.Register(path, body.ID, body.Name, body.PublicIP, body.Notes)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSONStatus(w, http.StatusOK, s)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleSecondaryByID(w http.ResponseWriter, r *http.Request) {
	path := inventory.DefaultFile
	rest := strings.TrimPrefix(r.URL.Path, "/api/secondaries/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "need id", 400)
		return
	}
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch {
	case r.Method == http.MethodGet && action == "":
		s, err := inventory.Get(path, id)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSONStatus(w, http.StatusOK, s)

	case r.Method == http.MethodPost && action == "validate":
		vr, err := svcpaths.PeerValidateLocal(id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		code := http.StatusOK
		if !vr.OK {
			code = http.StatusConflict // 409 before body
		}
		writeJSONStatus(w, code, vr)

	case r.Method == http.MethodPost && action == "promote":
		if err := svcpaths.PeerPromote(id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// Inventory may not have this id (plane-only peer); sync best-effort with explicit warning.
		invErr := inventory.UpdateLifecycle(path, id, "active", id, "")
		resp := map[string]any{"status": "promoted", "id": id, "plane_ok": true}
		if invErr != nil {
			resp["inventory_ok"] = false
			resp["inventory_error"] = invErr.Error()
			writeJSONStatus(w, http.StatusOK, resp) // plane succeeded; client must see inventory drift
			return
		}
		resp["inventory_ok"] = true
		writeJSONStatus(w, http.StatusOK, resp)

	case r.Method == http.MethodPost && action == "rollback":
		to := r.URL.Query().Get("to")
		if err := svcpaths.PeerRollback(to); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		resp := map[string]any{"status": "rollback", "to": to, "plane_ok": true}
		if to != "" {
			if err := inventory.UpdateLifecycle(path, to, "active", to, ""); err != nil {
				resp["inventory_ok"] = false
				resp["inventory_error"] = err.Error()
			} else {
				resp["inventory_ok"] = true
			}
		}
		writeJSONStatus(w, http.StatusOK, resp)

	case r.Method == http.MethodPost && action == "remove":
		if err := svcpaths.PeerRemove(id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		resp := map[string]any{"status": "removed", "id": id, "plane_ok": true}
		if err := inventory.SoftRemove(path, id); err != nil {
			resp["inventory_ok"] = false
			resp["inventory_error"] = err.Error()
		} else {
			resp["inventory_ok"] = true
		}
		writeJSONStatus(w, http.StatusOK, resp)

	case r.Method == http.MethodDelete && action == "":
		if err := inventory.SoftRemove(path, id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSONStatus(w, http.StatusOK, map[string]string{"status": "removed", "id": id})

	default:
		http.Error(w, "not found", 404)
	}
}

func handleSvcPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", 405)
		return
	}
	idx, err := svcpaths.PeerList()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSONStatus(w, http.StatusOK, idx)
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
