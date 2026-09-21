// Package guest implements MAC allow-list + short session codes for OpenWrt guest Wi‑Fi.
package guest

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultGrant = 10 * time.Minute
	MaxGrant     = 24 * time.Hour
	CodeTTL      = 15 * time.Minute
	TokenTTL     = 5 * time.Minute
)

// Config is stored on the edge device.
type Config struct {
	Enabled    bool   `json:"enabled"`
	SSID       string `json:"ssid"`
	Hidden     bool   `json:"hidden"`
	PSK        string `json:"psk"`
	SubnetCIDR string `json:"subnet_cidr"` // e.g. 192.168.50.1/24
	Band       string `json:"band"`        // 2g | both
	DeskPIN    string `json:"desk_pin"`    // plaintext only at setup; prefer DeskPINHash
	DeskPINHash string `json:"desk_pin_hash,omitempty"`
	DeskPort   int    `json:"desk_port"`
	CaptivePort int   `json:"captive_port"`
}

// AllowEntry is one MAC with internet until ExpiresAt.
type AllowEntry struct {
	MAC       string    `json:"mac"`
	ExpiresAt time.Time `json:"expires_at"`
	Note      string    `json:"note,omitempty"`
}

// Session is a pending captive session (no internet yet).
type Session struct {
	Code      string    `json:"code"`
	MAC       string    `json:"mac"`
	Token     string    `json:"token"` // one-time grant token for QR
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Store is file-backed state for allow-list and sessions.
type Store struct {
	mu      sync.Mutex
	dir     string
	Allow   []AllowEntry `json:"allow"`
	Pending []Session    `json:"pending"`
}

func NormalizeMAC(mac string) string {
	mac = strings.ToLower(strings.TrimSpace(mac))
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}

func HashPIN(pin string) string {
	sum := sha256.Sum256([]byte("netductor-guest-desk:" + pin))
	return hex.EncodeToString(sum[:])
}

func (c *Config) PINOK(pin string) bool {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return false
	}
	if c.DeskPINHash != "" {
		want, _ := hex.DecodeString(c.DeskPINHash)
		got := sha256.Sum256([]byte("netductor-guest-desk:" + pin))
		return subtle.ConstantTimeCompare(want, got[:]) == 1
	}
	if c.DeskPIN != "" {
		return subtle.ConstantTimeCompare([]byte(c.DeskPIN), []byte(pin)) == 1
	}
	return false
}

func ClampGrant(d time.Duration) time.Duration {
	if d <= 0 {
		return DefaultGrant
	}
	if d > MaxGrant {
		return MaxGrant
	}
	return d
}

func OpenStore(dir string) (*Store, error) {
	_ = os.MkdirAll(dir, 0o700)
	s := &Store{dir: dir}
	_ = s.load()
	return s, nil
}

func (s *Store) path() string {
	return filepath.Join(s.dir, "guest-state.json")
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path())
	if err != nil {
		return err
	}
	return json.Unmarshal(b, s)
}

func (s *Store) save() error {
	s.pruneLocked(time.Now())
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), append(b, '\n'), 0o600)
}

func (s *Store) pruneLocked(now time.Time) {
	var allow []AllowEntry
	for _, e := range s.Allow {
		if e.ExpiresAt.After(now) {
			allow = append(allow, e)
		}
	}
	s.Allow = allow
	var pend []Session
	for _, p := range s.Pending {
		if p.ExpiresAt.After(now) {
			pend = append(pend, p)
		}
	}
	s.Pending = pend
}

// EnsureSession returns existing pending session for MAC or creates one.
func (s *Store) EnsureSession(mac string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mac = NormalizeMAC(mac)
	now := time.Now()
	s.pruneLocked(now)
	for _, p := range s.Pending {
		if p.MAC == mac {
			return p, nil
		}
	}
	code, err := randomCode(3)
	if err != nil {
		return Session{}, err
	}
	// unique among pending
	for i := 0; i < 10; i++ {
		clash := false
		for _, p := range s.Pending {
			if p.Code == code {
				clash = true
				break
			}
		}
		if !clash {
			break
		}
		code, _ = randomCode(3)
	}
	tok, err := randomToken(16)
	if err != nil {
		return Session{}, err
	}
	sess := Session{
		Code:      code,
		MAC:       mac,
		Token:     tok,
		CreatedAt: now,
		ExpiresAt: now.Add(CodeTTL),
	}
	s.Pending = append(s.Pending, sess)
	return sess, s.save()
}

func (s *Store) LookupCode(code string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code = strings.ToUpper(strings.TrimSpace(code))
	now := time.Now()
	s.pruneLocked(now)
	for _, p := range s.Pending {
		if p.Code == code {
			return p, true
		}
	}
	return Session{}, false
}

func (s *Store) LookupToken(tok string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok = strings.TrimSpace(tok)
	now := time.Now()
	s.pruneLocked(now)
	for _, p := range s.Pending {
		if p.Token == tok {
			return p, true
		}
	}
	return Session{}, false
}

// Grant adds/extends MAC for duration (clamped). Consumes matching pending session.
func (s *Store) Grant(mac string, d time.Duration, note string) (AllowEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mac = NormalizeMAC(mac)
	d = ClampGrant(d)
	now := time.Now()
	s.pruneLocked(now)
	exp := now.Add(d)
	found := false
	for i := range s.Allow {
		if s.Allow[i].MAC == mac {
			s.Allow[i].ExpiresAt = exp
			s.Allow[i].Note = note
			found = true
			break
		}
	}
	e := AllowEntry{MAC: mac, ExpiresAt: exp, Note: note}
	if !found {
		s.Allow = append(s.Allow, e)
	} else {
		for _, a := range s.Allow {
			if a.MAC == mac {
				e = a
				break
			}
		}
	}
	// drop pending for this mac
	var pend []Session
	for _, p := range s.Pending {
		if p.MAC != mac {
			pend = append(pend, p)
		}
	}
	s.Pending = pend
	return e, s.save()
}

func (s *Store) Revoke(mac string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mac = NormalizeMAC(mac)
	var allow []AllowEntry
	for _, e := range s.Allow {
		if e.MAC != mac {
			allow = append(allow, e)
		}
	}
	s.Allow = allow
	return s.save()
}

func (s *Store) IsAllowed(mac string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	mac = NormalizeMAC(mac)
	now := time.Now()
	s.pruneLocked(now)
	for _, e := range s.Allow {
		if e.MAC == mac && e.ExpiresAt.After(now) {
			return true
		}
	}
	return false
}

func (s *Store) Snapshot() (allow []AllowEntry, pending []Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	allow = append([]AllowEntry(nil), s.Allow...)
	pending = append([]Session(nil), s.Pending...)
	return
}

func randomCode(n int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range b {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out), nil
}

func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// JoinQR returns WIFI: QR payload (WPA, optional hidden).
func JoinQR(ssid, psk string, hidden bool) string {
	// WIFI:T:WPA;S:ssid;P:pass;H:true;;
	h := "false"
	if hidden {
		h = "true"
	}
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `;`, `\;`)
		s = strings.ReplaceAll(s, `,`, `\,`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		return s
	}
	return fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;H:%s;;", esc(ssid), esc(psk), h)
}

func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.DeskPort == 0 {
		c.DeskPort = 7880
	}
	if c.CaptivePort == 0 {
		c.CaptivePort = 7881
	}
	if c.SSID == "" {
		c.SSID = "Guest"
	}
	if c.SubnetCIDR == "" {
		c.SubnetCIDR = "192.168.50.1/24"
	}
	if c.Band == "" {
		c.Band = "2g"
	}
	return c, nil
}

func SaveConfig(path string, c Config) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	if c.DeskPIN != "" && c.DeskPINHash == "" {
		c.DeskPINHash = HashPIN(c.DeskPIN)
		c.DeskPIN = "" // don't leave plaintext
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}
