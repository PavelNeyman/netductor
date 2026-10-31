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

// SaveJSON writes v atomically (CreateTemp + rename), mode 0600, parent dir 0700.
func SaveJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".nd-state-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	data := append(raw, '\n')
	n, err := f.Write(data)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write state file: %w", err)
	}
	if n != len(data) {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write state file: short write (%d of %d bytes)", n, len(data))
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("sync state file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close state file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace state file: %w", err)
	}
	df, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open state directory: %w", err)
	}
	if err := df.Sync(); err != nil {
		_ = df.Close()
		return fmt.Errorf("sync state directory: %w", err)
	}
	if err := df.Close(); err != nil {
		return fmt.Errorf("close state directory: %w", err)
	}
	return nil
}

// IsNotExist reports whether err is missing-file semantics.
func IsNotExist(err error) bool {
	return errors.Is(err, ErrNotExist) || os.IsNotExist(err)
}
