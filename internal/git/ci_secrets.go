package gitstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// CISecretsDir is /var/lib/netductor/ci-secrets/<project>/
func CISecretsDir(project string) string {
	return filepath.Join(paths.StateDir(), "ci-secrets", sanitize(project))
}

// LoadCISecretsEnv returns KEY=value from files in secrets dir (filename = key, content = value).
// Values never logged by caller.
func LoadCISecretsEnv(project string) []string {
	dir := CISecretsDir(project)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		key := e.Name()
		if !validSecretKey(key) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, key))
		if err != nil {
			continue
		}
		val := strings.TrimRight(string(b), "\r\n")
		out = append(out, key+"="+val)
	}
	return out
}

func validSecretKey(k string) bool {
	if k == "" || len(k) > 64 {
		return false
	}
	for _, c := range k {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}

// SetCISecret writes a secret file (0600).
func SetCISecret(project, key, value string) error {
	key = strings.TrimSpace(key)
	if !validSecretKey(key) {
		return fmt.Errorf("invalid secret key")
	}
	dir := CISecretsDir(project)
	_ = os.MkdirAll(dir, 0o700)
	return os.WriteFile(filepath.Join(dir, key), []byte(value), 0o600)
}

// ListCISecretKeys names only.
func ListCISecretKeys(project string) ([]string, error) {
	ents, err := os.ReadDir(CISecretsDir(project))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var keys []string
	for _, e := range ents {
		if !e.IsDir() && validSecretKey(e.Name()) {
			keys = append(keys, e.Name())
		}
	}
	return keys, nil
}

// DeleteCISecret removes one key.
func DeleteCISecret(project, key string) error {
	if !validSecretKey(key) {
		return fmt.Errorf("invalid key")
	}
	return os.Remove(filepath.Join(CISecretsDir(project), key))
}
