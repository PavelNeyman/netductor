package gitstore

import (
	"net/url"
	"os"
	"strings"
)

func githubTokenLocal() string {
	// Prefer dedicated repos token, then shared releases token / env.
	if v := os.Getenv("NETDUCTOR_GITHUB_TOKEN_REPOS"); v != "" {
		return strings.TrimSpace(v)
	}
	for _, p := range []string{"/etc/netductor/secrets/github_token_repos"} {
		b, err := os.ReadFile(p)
		if err == nil {
			if t := strings.TrimSpace(string(b)); t != "" {
				return t
			}
		}
	}
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
			if t := strings.TrimSpace(string(b)); t != "" {
				return t
			}
		}
	}
	return ""
}

// withGitHubAuth embeds token for private HTTPS GitHub remotes.
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
	if u.User != nil && u.User.Username() != "" {
		return raw
	}
	u.User = url.UserPassword("x-access-token", tok)
	return u.String()
}
