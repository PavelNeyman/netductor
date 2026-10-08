package update

import (
	"os"
	"path/filepath"
	"strings"
)

const TokenFile = "/etc/netductor/secrets/github_token"
const TokenFileRepos = "/etc/netductor/secrets/github_token_repos"

// TokenKind selects which secret file.
const (
	TokenKindReleases = "releases" // list/download netductor GH releases
	TokenKindRepos   = "repos"    // private git mirror / migrate
)

// TokenConfigured reports whether a GitHub token is available (file or env).
func TokenConfigured() bool {
	return githubToken() != ""
}

// TokenStatus for UI (never returns raw token).
type TokenStatus struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source,omitempty"` // file | env | none
	Hint       string `json:"hint,omitempty"`   // masked prefix
	Kind       string `json:"kind,omitempty"`   // releases | repos
}

// TokensStatus both slots for dual-token UI.
type TokensStatus struct {
	Releases TokenStatus `json:"releases"`
	Repos    TokenStatus `json:"repos"`
}

func GetTokenStatus() TokenStatus {
	st := statusFor(TokenFile, []string{"NETDUCTOR_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"})
	st.Kind = TokenKindReleases
	return st
}

func GetReposTokenStatus() TokenStatus {
	st := statusFor(TokenFileRepos, []string{"NETDUCTOR_GITHUB_TOKEN_REPOS"})
	st.Kind = TokenKindRepos
	// fallback display: if repos empty but releases set, note inheritance in Source
	if !st.Configured {
		if r := GetTokenStatus(); r.Configured {
			st.Source = "fallback:releases"
			st.Hint = r.Hint
			// Configured stays false for "dedicated" but UI can show fallback
		}
	}
	return st
}

func GetTokensStatus() TokensStatus {
	return TokensStatus{Releases: GetTokenStatus(), Repos: GetReposTokenStatus()}
}

func statusFor(file string, envKeys []string) TokenStatus {
	if b, err := os.ReadFile(file); err == nil {
		t := strings.TrimSpace(string(b))
		if t != "" {
			return TokenStatus{Configured: true, Source: "file", Hint: maskToken(t)}
		}
	}
	for _, k := range envKeys {
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

// SetToken writes releases token (backward compatible).
func SetToken(token string) error {
	return SetTokenKind(TokenKindReleases, token)
}

func SetTokenKind(kind, token string) error {
	token = strings.TrimSpace(token)
	path := TokenFile
	if kind == TokenKindRepos {
		path = TokenFileRepos
	}
	if token == "" {
		return ClearTokenKind(kind)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}

func ClearToken() error {
	return ClearTokenKind(TokenKindReleases)
}

func ClearTokenKind(kind string) error {
	path := TokenFile
	if kind == TokenKindRepos {
		path = TokenFileRepos
	}
	_ = os.Remove(path)
	return nil
}

// githubTokenReleases for API/releases (env overrides file).
func githubToken() string {
	for _, k := range []string{"NETDUCTOR_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if t := strings.TrimSpace(os.Getenv(k)); t != "" {
			return t
		}
	}
	if b, err := os.ReadFile(TokenFile); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			return t
		}
	}
	return ""
}

// GithubTokenRepos for private git mirrors; falls back to releases token.
func GithubTokenRepos() string {
	if t := strings.TrimSpace(os.Getenv("NETDUCTOR_GITHUB_TOKEN_REPOS")); t != "" {
		return t
	}
	if b, err := os.ReadFile(TokenFileRepos); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			return t
		}
	}
	return githubToken()
}

// GithubTokenForAPI is releases token (exported).
func GithubTokenForAPI() string { return githubToken() }
