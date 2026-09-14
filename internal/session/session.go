package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Production defaults (not home-lab): short-lived operator sessions.
const (
	DefaultHours = 8
	MaxHours     = 72
	TokenBytes   = 32 // 256-bit
)

type Meta struct {
	Exp     int64  `json:"exp"`
	Created int64  `json:"created"`
	Label   string `json:"label,omitempty"`
	IP      string `json:"ip,omitempty"`
}

var mu sync.Mutex

func Dir() string { return paths.SessionsDir() }

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func pathForHash(h string) string {
	return filepath.Join(Dir(), h+".json")
}

func TokenFromAuth(auth, cookie string) string {
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	for _, p := range strings.Split(cookie, ";") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "nd_session=") {
			if i := strings.Index(p, "="); i >= 0 {
				return strings.TrimSpace(p[i+1:])
			}
		}
	}
	return ""
}

func clampHours(hours int) int {
	if hours <= 0 {
		hours = DefaultHours
	}
	if hours > MaxHours {
		hours = MaxHours
	}
	return hours
}

// Create issues a new session. Returns plaintext token once; only hash is stored.
func Create(hours int, label, clientIP string) (token string, exp int64, err error) {
	hours = clampHours(hours)
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(Dir(), 0o700)
	var b [TokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", 0, err
	}
	token = hex.EncodeToString(b[:])
	now := time.Now().Unix()
	exp = now + int64(hours)*3600
	meta := Meta{Exp: exp, Created: now, Label: label, IP: clientIP}
	raw, _ := json.Marshal(meta)
	if err := os.WriteFile(pathForHash(hashToken(token)), raw, 0o600); err != nil {
		return "", 0, err
	}
	return token, exp, nil
}

func loadMeta(token string) (Meta, bool) {
	if token == "" || len(token) > 128 || strings.Contains(token, "/") || strings.Contains(token, "..") {
		return Meta{}, false
	}
	// support legacy plaintext-filename sessions during one release
	legacy := filepath.Join(Dir(), token)
	if b, err := os.ReadFile(legacy); err == nil {
		var exp int64
		fmt.Sscanf(strings.TrimSpace(string(b)), "%d", &exp)
		if exp >= time.Now().Unix() {
			return Meta{Exp: exp}, true
		}
		_ = os.Remove(legacy)
		return Meta{}, false
	}
	b, err := os.ReadFile(pathForHash(hashToken(token)))
	if err != nil {
		return Meta{}, false
	}
	var m Meta
	if json.Unmarshal(b, &m) != nil || m.Exp < time.Now().Unix() {
		_ = os.Remove(pathForHash(hashToken(token)))
		return Meta{}, false
	}
	return m, true
}

func Expiry(token string) (int64, bool) {
	mu.Lock()
	defer mu.Unlock()
	m, ok := loadMeta(token)
	if !ok {
		return 0, false
	}
	return m.Exp, true
}

func Valid(token string) bool {
	_, ok := Expiry(token)
	return ok
}

func Revoke(token string) {
	mu.Lock()
	defer mu.Unlock()
	_ = os.Remove(pathForHash(hashToken(token)))
	_ = os.Remove(filepath.Join(Dir(), token)) // legacy
}

func RevokeAll() error {
	mu.Lock()
	defer mu.Unlock()
	ents, err := os.ReadDir(Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range ents {
		_ = os.Remove(filepath.Join(Dir(), e.Name()))
	}
	return nil
}


// SessionInfo is a redacted view (no plaintext token).
type SessionInfo struct {
	HashPrefix string `json:"id"`
	Exp        int64  `json:"exp"`
	Created    int64  `json:"created,omitempty"`
	Label      string `json:"label,omitempty"`
	IP         string `json:"ip,omitempty"`
}

// List returns active (non-expired) sessions from disk.
func List() []SessionInfo {
	mu.Lock()
	defer mu.Unlock()
	ents, err := os.ReadDir(Dir())
	if err != nil {
		return nil
	}
	now := time.Now().Unix()
	var out []SessionInfo
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		b, err := os.ReadFile(filepath.Join(Dir(), name))
		if err != nil {
			continue
		}
		var m Meta
		if json.Unmarshal(b, &m) != nil {
			// legacy exp-only
			var exp int64
			fmt.Sscanf(strings.TrimSpace(string(b)), "%d", &exp)
			m.Exp = exp
		}
		if m.Exp < now {
			_ = os.Remove(filepath.Join(Dir(), name))
			continue
		}
		pref := name
		if len(pref) > 12 {
			pref = pref[:12]
		}
		out = append(out, SessionInfo{HashPrefix: pref, Exp: m.Exp, Created: m.Created, Label: m.Label, IP: m.IP})
	}
	return out
}
