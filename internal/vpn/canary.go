package vpn

import (
	"encoding/json"
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
	_ = os.MkdirAll(filepath.Dir(canaryPath()), 0o700)
	c := CanaryConfig{Users: users}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(canaryPath(), append(b, '\n'), 0o600)
}
