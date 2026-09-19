package nvr

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type clipToken struct {
	Token    string `json:"token"`
	Path     string `json:"path"`
	CameraID string `json:"camera_id"`
	Exp      int64  `json:"exp"`
}

var (
	tokMu sync.Mutex
)

func tokensPath() string {
	return filepath.Join(dir(), "clip_tokens.json")
}

func loadTokens() []clipToken {
	var list []clipToken
	b, err := os.ReadFile(tokensPath())
	if err != nil {
		return list
	}
	_ = json.Unmarshal(b, &list)
	return list
}

func saveTokens(list []clipToken) error {
	_ = ensure()
	// drop expired
	now := Now().Unix()
	out := list[:0]
	for _, t := range list {
		if t.Exp > now {
			out = append(out, t)
		}
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	tmp := tokensPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, tokensPath())
}

// IssueClipToken creates a short-lived token for one segment path (TTL seconds).
func IssueClipToken(cameraID, absPath string, ttlSec int) (string, error) {
	if ttlSec <= 0 {
		ttlSec = 120
	}
	if ttlSec > 3600 {
		ttlSec = 3600
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	tokMu.Lock()
	defer tokMu.Unlock()
	list := loadTokens()
	list = append(list, clipToken{
		Token: tok, Path: absPath, CameraID: cameraID, Exp: Now().Add(time.Duration(ttlSec) * time.Second).Unix(),
	})
	if err := saveTokens(list); err != nil {
		return "", err
	}
	return tok, nil
}

// RedeemClipToken returns path if valid and removes token (one-shot).
func RedeemClipToken(token string) (path string, ok bool) {
	tokMu.Lock()
	defer tokMu.Unlock()
	list := loadTokens()
	now := Now().Unix()
	out := list[:0]
	found := ""
	for _, t := range list {
		if t.Token == token && t.Exp > now && found == "" {
			found = t.Path
			continue // consume
		}
		if t.Exp > now {
			out = append(out, t)
		}
	}
	_ = saveTokens(out)
	if found == "" {
		return "", false
	}
	return found, true
}
