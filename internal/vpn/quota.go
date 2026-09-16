package vpn

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SoftLimitGB reads optional soft limit (GiB) for user from state/vpn_quota.json
// {"operator": 200}. Zero or missing = no limit. Enforcement is alert-only (soft).
func SoftLimitGB(name string) float64 {
	b, err := os.ReadFile(filepath.Join(paths.StateDir(), "vpn_quota.json"))
	if err != nil {
		return 0
	}
	var m map[string]float64
	if json.Unmarshal(b, &m) != nil {
		return 0
	}
	return m[name]
}

func SetSoftLimitGB(name string, gb float64) error {
	path := filepath.Join(paths.StateDir(), "vpn_quota.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	m := map[string]float64{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	if gb <= 0 {
		delete(m, name)
	} else {
		m[name] = gb
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}
