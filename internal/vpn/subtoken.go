package vpn

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type subEntry struct {
	Token     string    `json:"token"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type subStore struct {
	Entries []subEntry `json:"entries"`
}

var subMu sync.Mutex

func subPath() string {
	return filepath.Join(paths.StateDir(), "vpn_sub_tokens.json")
}

func loadSub() subStore {
	var s subStore
	b, err := os.ReadFile(subPath())
	if err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func saveSub(s subStore) error {
	_ = os.MkdirAll(filepath.Dir(subPath()), 0o700)
	raw, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(subPath(), append(raw, '\n'), 0o600)
}

// IssueSubToken creates or rotates a long random token for user subscription URL.
func IssueSubToken(user string) (string, error) {
	if !ValidName(user) {
		return "", fmt.Errorf("invalid user")
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(buf)
	subMu.Lock()
	defer subMu.Unlock()
	s := loadSub()
	// remove old for user
	out := s.Entries[:0]
	for _, e := range s.Entries {
		if e.User != user {
			out = append(out, e)
		}
	}
	out = append(out, subEntry{Token: tok, User: user, CreatedAt: time.Now().UTC()})
	s.Entries = out
	if err := saveSub(s); err != nil {
		return "", err
	}
	return tok, nil
}

// ResolveSubToken returns user name for token or empty.
func ResolveSubToken(token string) string {
	if len(token) < 16 {
		return ""
	}
	subMu.Lock()
	defer subMu.Unlock()
	for _, e := range loadSub().Entries {
		if e.Token == token {
			return e.User
		}
	}
	return ""
}

// SubTokenForUser returns existing token or empty.
func SubTokenForUser(user string) string {
	subMu.Lock()
	defer subMu.Unlock()
	for _, e := range loadSub().Entries {
		if e.User == user {
			return e.Token
		}
	}
	return ""
}

// RevokeSubToken removes token for user.
func RevokeSubToken(user string) error {
	subMu.Lock()
	defer subMu.Unlock()
	s := loadSub()
	out := s.Entries[:0]
	for _, e := range s.Entries {
		if e.User != user {
			out = append(out, e)
		}
	}
	s.Entries = out
	return saveSub(s)
}

// EnsureSubToken returns existing or issues new.
func EnsureSubToken(user string) (string, error) {
	if t := SubTokenForUser(user); t != "" {
		return t, nil
	}
	return IssueSubToken(user)
}
