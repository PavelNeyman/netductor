package operator

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func handleSessionLocal(opToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", 405)
			return
		}
		if !requireToken(r, opToken) {
			http.Error(w, "unauthorized", 401)
			return
		}
		home, _ := os.UserHomeDir()
		p := filepath.Join(home, ".netductor", "node_session")
		b, err := os.ReadFile(p)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "path": p})
			return
		}
		tok := strings.TrimSpace(string(b))
		if i := strings.IndexByte(tok, '\n'); i >= 0 {
			tok = strings.TrimSpace(tok[:i])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "token": tok, "path": p,
		})
	}
}
