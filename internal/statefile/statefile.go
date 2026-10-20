// Package statefile is a small durable JSON load/save helper (NotExist vs error, atomic write).
package statefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotExist is returned when the file is missing (clean install).
var ErrNotExist = errors.New("statefile: not exist")

// LoadJSON reads path into dest. Missing file → ErrNotExist (dest unchanged).
// Corrupt JSON or I/O errors are returned as-is.
func LoadJSON(path string, dest any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotExist
		}
		return err
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("statefile %s: %w", path, err)
	}
	return nil
}

// SaveJSON writes v atomically (tmp + rename), mode 0600, parent dir 0700.
func SaveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// IsNotExist reports whether err is missing-file semantics.
func IsNotExist(err error) bool {
	return errors.Is(err, ErrNotExist) || os.IsNotExist(err)
}
