package operator

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/inventory"
)

func inventoryPath() string {
	if d := os.Getenv("NETDUCTOR_INVENTORY"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".netductor", "inventory", "secondaries.json")
}

func handleSecondariesInventory(w http.ResponseWriter, r *http.Request, token string) {
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	path := inventoryPath()
	switch r.Method {
	case http.MethodGet:
		items, err := inventory.List(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSONOp(w, 200, items)
	case http.MethodPost:
		var body struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			PublicIP string `json:"public_ip"`
			Notes    string `json:"notes"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		s, err := inventory.Register(path, body.ID, body.Name, body.PublicIP, body.Notes)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSONOp(w, 200, s)
	default:
		http.Error(w, "GET or POST", 405)
	}
}

func handleSecondaryInventoryID(w http.ResponseWriter, r *http.Request, token string) {
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	path := inventoryPath()
	rest := strings.TrimPrefix(r.URL.Path, "/v1/secondaries/")
	id := strings.Split(strings.Trim(rest, "/"), "/")[0]
	if id == "" {
		http.Error(w, "need id", 400)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s, err := inventory.Get(path, id)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSONOp(w, 200, s)
	case http.MethodDelete:
		if err := inventory.SoftRemove(path, id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSONOp(w, 200, map[string]string{"status": "removed", "id": id})
	default:
		http.Error(w, "GET or DELETE", 405)
	}
}

