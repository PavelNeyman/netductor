package update

import (
	"os"
	"path/filepath"
	"strings"
)

const TokenFile = "/etc/netductor/secrets/github_token"

// TokenConfigured reports whether a GitHub token is available (file or env).
func TokenConfigured() bool {
	return githubToken() != ""
}

// TokenStatus for UI (never returns raw token).
type TokenStatus struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source,omitempty"` // file | env | none
	Hint       string `json:"hint,omitempty"`   // masked prefix
}

func GetTokenStatus() TokenStatus {
	if b, err := os.ReadFile(TokenFile); err == nil {
		t := strings.TrimSpace(string(b))
		if t != "" {
			return TokenStatus{Configured: true, Source: "file", Hint: maskToken(t)}
		}
	}
	for _, k := range []string{"NETDUCTOR_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if t := strings.TrimSpace(os.Getenv(k)); t != "" {
			return TokenStatus{Configured: true, Source: "env:" + k, Hint: maskToken(t)}
		}
	}
	return TokenStatus{Configured: false, Source: "none"}
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return "****"
	}
	return t[:4] + "…" + t[len(t)-4:]
}

// SetToken writes token to secrets file (0600). Empty clears.
func SetToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ClearToken()
	}
	if err := os.MkdirAll(filepath.Dir(TokenFile), 0o700); err != nil {
		return err
	}
	return os.WriteFile(TokenFile, []byte(token+"\n"), 0o600)
}

// ClearToken removes the secrets file (env vars unaffected).
func ClearToken() error {
	_ = os.Remove(TokenFile)
	return nil
}
