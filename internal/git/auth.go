package gitstore

import (
	"net/url"
	"os"
	"strings"
)

func githubTokenLocal() string {
	if v := os.Getenv("NETDUCTOR_GITHUB_TOKEN"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := os.Getenv("GH_TOKEN"); v != "" {
		return strings.TrimSpace(v)
	}
	for _, p := range []string{"/etc/netductor/secrets/github_token", "/etc/netductor/github_token"} {
		b, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// withGitHubAuth embeds token for private HTTPS GitHub remotes.
// Prefer x-access-token for PATs. Non-GitHub URLs unchanged.
func withGitHubAuth(raw string) string {
	tok := githubTokenLocal()
	if tok == "" || raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return raw
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return raw
	}
	// already has userinfo
	if u.User != nil && u.User.Username() != "" {
		return raw
	}
	u.User = url.UserPassword("x-access-token", tok)
	return u.String()
}
