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
		writeJSONBody(w, items)
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
		writeJSONBody(w, s)
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
		writeJSONBody(w, s)
	case r.Method == http.MethodPost && action == "validate":
		vr, err := svcpaths.PeerValidateLocal(id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSONBody(w, vr)
		if !vr.OK {
			w.WriteHeader(409)
		}
	case r.Method == http.MethodPost && action == "promote":
		if err := svcpaths.PeerPromote(id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = inventory.UpdateLifecycle(path, id, "active", id, "")
		writeJSONBody(w, map[string]string{"status": "promoted", "id": id})
	case r.Method == http.MethodPost && action == "rollback":
		to := r.URL.Query().Get("to")
		if err := svcpaths.PeerRollback(to); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSONBody(w, map[string]string{"status": "rollback", "to": to})
	case r.Method == http.MethodPost && action == "remove":
		if err := svcpaths.PeerRemove(id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		_ = inventory.SoftRemove(path, id)
		writeJSONBody(w, map[string]string{"status": "removed", "id": id})
	case r.Method == http.MethodDelete && action == "":
		if err := inventory.SoftRemove(path, id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSONBody(w, map[string]string{"status": "removed", "id": id})
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
	writeJSONBody(w, idx)
}

func writeJSONBody(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
