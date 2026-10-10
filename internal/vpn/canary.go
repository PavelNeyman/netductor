package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// CanaryConfig lists VPN user names treated as "our" clients for mismatch alerts.
type CanaryConfig struct {
	Users []string `json:"users"`
}

func canaryPath() string {
	return filepath.Join(paths.EtcDir(), "canary-users.json")
}

// LoadCanaryUsers returns configured canary names (default: empty = use heuristic later).
func LoadCanaryUsers() []string {
	b, err := os.ReadFile(canaryPath())
	if err != nil {
		return nil
	}
	var c CanaryConfig
	if json.Unmarshal(b, &c) != nil {
		return nil
	}
	var out []string
	for _, u := range c.Users {
		u = strings.TrimSpace(u)
		if u != "" && u != "relay-uplink" {
			out = append(out, u)
		}
	}
	return out
}

// SaveCanaryUsers writes list.
func SaveCanaryUsers(users []string) error {
	users = FilterKnownVPNUsers(users)
	_ = os.MkdirAll(filepath.Dir(canaryPath()), 0o700)
	c := CanaryConfig{Users: users}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(canaryPath(), append(b, '\n'), 0o600)
}


// IsCanary reports whether name is in the canary list.
func IsCanary(name string) bool {
	name = strings.TrimSpace(name)
	for _, u := range LoadCanaryUsers() {
		if u == name {
			return true
		}
	}
	return false
}

// ToggleCanary adds or removes name; returns new membership.
func ToggleCanary(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "relay-uplink" || IsEdgeUser(name) {
		return false, fmt.Errorf("invalid user")
	}
	if filtered := FilterKnownVPNUsers([]string{name}); len(filtered) == 0 {
		// registry available and name unknown
		if users, err := ListNative(); err == nil && len(users) > 0 {
			return false, fmt.Errorf("unknown vpn user: %s", name)
		}
	}
	cur := LoadCanaryUsers()
	var next []string
	found := false
	for _, u := range cur {
		if u == name {
			found = true
			continue
		}
		next = append(next, u)
	}
	if !found {
		next = append(next, name)
	}
	if err := SaveCanaryUsers(next); err != nil {
		return false, err
	}
	return !found, nil
}


// EnsureCanarySeed writes canary-users.json once if missing, with non-system VPN users.
func EnsureCanarySeed() error {
	if _, err := os.Stat(canaryPath()); err == nil {
		return nil
	}
	users, err := ListNative()
	if err != nil {
		return err
	}
	var names []string
	for _, u := range users {
		if u.Name == "" || u.Name == "relay-uplink" || IsEdgeUser(u.Name) {
			continue
		}
		names = append(names, u.Name)
	}
	if len(names) == 0 {
		return nil
	}
	return SaveCanaryUsers(names)
}


// FilterKnownVPNUsers drops names not in vpn-users (and system accounts).
func FilterKnownVPNUsers(names []string) []string {
	users, err := ListNative()
	known := map[string]bool{}
	if err == nil {
		for _, u := range users {
			if u.Name != "" && u.Name != "relay-uplink" && !IsEdgeUser(u.Name) {
				known[u.Name] = true
			}
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		if len(known) > 0 && !known[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}
