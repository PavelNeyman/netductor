package operator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func handleSessionIssue(opToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		if !requireToken(r, opToken) {
			http.Error(w, "unauthorized", 401)
			return
		}
		var body struct {
			Host  string `json:"host"`
			User  string `json:"user"`
			Key   string `json:"key"`
			Hours string `json:"hours"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		host := strings.TrimSpace(body.Host)
		if host == "" {
			http.Error(w, "host required", 400)
			return
		}
		user := orDefault(body.User, "root")
		key := expandKeyPath(body.Key)
		hours := orDefault(body.Hours, "72")
		sshPort := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
		if sshPort == "" {
			sshPort = "52222"
		}
		remote := fmt.Sprintf("netductor vpn session %s", hours)
		cmd := exec.Command("ssh", "-p", sshPort, "-i", key, "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new",
			"--", user+"@"+host, remote)
		out, err := cmd.Output()
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			msg := err.Error()
			if ee, ok := err.(*exec.ExitError); ok {
				msg = strings.TrimSpace(string(ee.Stderr)) + " " + msg
			}
			w.WriteHeader(500)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
			return
		}
		tok := strings.TrimSpace(string(out))
		if i := strings.IndexByte(tok, '\n'); i >= 0 {
			tok = strings.TrimSpace(tok[:i])
		}
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".netductor")
		_ = os.MkdirAll(dir, 0o700)
		_ = os.WriteFile(filepath.Join(dir, "node_session"), []byte(tok+"\n"), 0o600)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "token": tok, "saved": filepath.Join(dir, "node_session")})
	}
}
