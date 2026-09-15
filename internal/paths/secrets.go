package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadSecret returns secret file contents. Tries name then legacyName under secrets/.
func ReadSecret(name string, legacyNames ...string) string {
	dir := filepath.Join(EtcDir(), "secrets")
	for _, n := range append([]string{name}, legacyNames...) {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				return v
			}
		}
	}
	return ""
}

// WriteSecret writes secret with mode 0600.
func WriteSecret(name, value string) error {
	dir := filepath.Join(EtcDir(), "secrets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), append([]byte(strings.TrimSpace(value)), '\n'), 0o600)
}
