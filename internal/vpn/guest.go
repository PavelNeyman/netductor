package vpn

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type Guest struct {
	ID        string `json:"id"`
	User      string `json:"user"` // vpn user name created for guest
	Expires   int64  `json:"expires"`
	Created   int64  `json:"created"`
	Note      string `json:"note,omitempty"`
}

func guestPath() string {
	return filepath.Join(paths.StateDir(), "vpn_guests.json")
}

func loadGuests() ([]Guest, error) {
	b, err := os.ReadFile(guestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var g []Guest
	_ = json.Unmarshal(b, &g)
	return g, nil
}

func saveGuests(g []Guest) error {
	_ = os.MkdirAll(filepath.Dir(guestPath()), 0o700)
	raw, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(guestPath(), append(raw, '\n'), 0o600)
}

// CreateGuest registers metadata; caller should AddUser first.
// EnsureGuestUser returns the shared "guest" VPN account (creates if missing).
func EnsureGuestUser() (string, error) {
	users, _ := List()
	for _, u := range users {
		if u.Name == "guest" {
			return "guest", nil
		}
	}
	_, err := Add("guest", "shared guest access")
	return "guest", err
}

// IssueGuestAccess creates/rotates guest credentials window with TTL.
// Same user "guest"; expires → Revoke disables until next issue.
func IssueGuestAccess(ttl time.Duration, note string) (Guest, string, error) {
	name, err := EnsureGuestUser()
	if err != nil {
		return Guest{}, "", err
	}
	_, _ = Enable(name)
	g, err := CreateGuest(name, note, ttl)
	if err != nil {
		return g, "", err
	}
	// preferred vless link
	link := ""
	if u, ok := ReadClient(name, "vless", "link"); ok {
		link = u
	}
	return g, link, nil
}

func CreateGuest(user, note string, ttl time.Duration) (Guest, error) {
	if ttl < time.Minute {
		ttl = 10 * time.Minute
	}
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	g := Guest{
		ID:      hex.EncodeToString(id),
		User:    user,
		Expires: time.Now().Add(ttl).Unix(),
		Created: time.Now().Unix(),
		Note:    note,
	}
	list, _ := loadGuests()
	list = append(list, g)
	return g, saveGuests(list)
}

// ExpireGuests removes expired guest users (DeleteUser) and metadata.
func ExpireGuests() (int, error) {
	list, err := loadGuests()
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	var keep []Guest
	n := 0
	for _, g := range list {
		if g.Expires > 0 && g.Expires < now {
			_, _ = Revoke(g.User)
			n++
			continue
		}
		keep = append(keep, g)
	}
	if err := saveGuests(keep); err != nil {
		return n, err
	}
	return n, nil
}

func ListGuests() []Guest {
	g, _ := loadGuests()
	return g
}

func GuestSummary() string {
	g := ListGuests()
	if len(g) == 0 {
		return "guests: 0"
	}
	return fmt.Sprintf("guests: %d", len(g))
}
