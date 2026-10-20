package vpn

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/statefile"
	"github.com/PavelNeyman/netductor/internal/secreq"
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

func loadSub() (subStore, error) {
	var s subStore
	err := statefile.LoadJSON(subPath(), &s)
	if statefile.IsNotExist(err) {
		return s, nil
	}
	return s, err
}

func saveSub(s subStore) error {
	return statefile.SaveJSON(subPath(), s)
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
	s, err := loadSub()
	if err != nil {
		return "", err
	}
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
	s, err := loadSub()
	if err != nil {
		return ""
	}
	for _, e := range s.Entries {
		if secreq.Eq(e.Token, token) {
			return e.User
		}
	}
	return ""
}

// SubTokenForUser returns existing token or empty.
func SubTokenForUser(user string) string {
	subMu.Lock()
	defer subMu.Unlock()
	s, err := loadSub()
	if err != nil {
		return ""
	}
	for _, e := range s.Entries {
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
	s, err := loadSub()
	if err != nil {
		return err
	}
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

// RenameSubToken moves the subscription token from oldName to newName.
func RenameSubToken(oldName, newName string) error {
	subMu.Lock()
	defer subMu.Unlock()
	s, err := loadSub()
	if err != nil {
		return err
	}
	changed := false
	for i := range s.Entries {
		if s.Entries[i].User == oldName {
			s.Entries[i].User = newName
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return saveSub(s)
}

// ListOrphanSubUsers returns subscription token user names not present in live.
func ListOrphanSubUsers(live []string) []string {
	want := map[string]bool{}
	for _, n := range live {
		want[n] = true
	}
	subMu.Lock()
	defer subMu.Unlock()
	s, err := loadSub()
	if err != nil {
		return nil
	}
	var orphans []string
	for _, e := range s.Entries {
		if !want[e.User] {
			orphans = append(orphans, e.User)
		}
	}
	return orphans
}

// PruneOrphanSubTokens drops tokens whose user is not in the VPN registry.
// N-12: empty live list refuses prune when tokens exist (avoids wiping all on List failure).
func PruneOrphanSubTokens(live []string) (int, error) {
	want := map[string]bool{}
	for _, n := range live {
		want[n] = true
	}
	subMu.Lock()
	defer subMu.Unlock()
	s, err := loadSub()
	if err != nil {
		return 0, err
	}
	if len(live) == 0 && len(s.Entries) > 0 {
		return 0, fmt.Errorf("refuse prune: empty live user list with %d tokens (vpn list failed?)", len(s.Entries))
	}
	out := s.Entries[:0]
	n := 0
	for _, e := range s.Entries {
		if want[e.User] {
			out = append(out, e)
			continue
		}
		n++
	}
	if n == 0 {
		return 0, nil
	}
	s.Entries = out
	return n, saveSub(s)
}
