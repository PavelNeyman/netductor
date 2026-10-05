package operator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// sanitizeSessionHours allows only 1–168 digit hours for remote "vpn session <n>".
func sanitizeSessionHours(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "72", nil
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("hours must be digits only")
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 168 {
		return "", fmt.Errorf("hours must be 1–168")
	}
	return strconv.Itoa(n), nil
}

// sanitizeSSHTargetUserHost rejects shell metacharacters in user@host pieces.
func sanitizeSSHTargetUserHost(user, host string) (string, string, error) {
	user = strings.TrimSpace(user)
	host = strings.TrimSpace(host)
	if user == "" {
		user = "root"
	}
	if host == "" {
		return "", "", fmt.Errorf("host required")
	}
	for _, r := range user + host {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '.' || r == '-' || r == '_' || r == ':' {
			continue
		}
		return "", "", fmt.Errorf("invalid user/host character")
	}
	return user, host, nil
}

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
		user, host, err := sanitizeSSHTargetUserHost(orDefault(body.User, "root"), body.Host)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		key := expandKeyPath(body.Key)
		hours, err := sanitizeSessionHours(orDefault(body.Hours, "72"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		sshPort := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
		if sshPort == "" {
			sshPort = "52222"
		}
		remote := "netductor vpn session " + hours
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
		if i := strings.IndexByte(tok, 10); i >= 0 {
			tok = strings.TrimSpace(tok[:i])
		}
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".netductor")
		_ = os.MkdirAll(dir, 0o700)
		_ = os.WriteFile(filepath.Join(dir, "node_session"), []byte(tok+"\n"), 0o600)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "token": tok, "saved": filepath.Join(dir, "node_session")})
	}
}
