package nvr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveStoragePath returns the effective segments path for the configured backend.
// nfs: Path must already be the mountpoint (operator mounts via systemd/fstab).
// local / local_encrypted: Path as configured under state dir or custom.
func ResolveStoragePath() (string, error) {
	c := LoadConfig()
	p := strings.TrimSpace(c.Path)
	if p == "" {
		p = defaultConfig().Path
	}
	switch strings.ToLower(c.StorageBackend) {
	case "nfs", "home_nfs", "home":
		// Expect operator-mounted NFS; we only verify writable.
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return p, fmt.Errorf("nfs path %s not mounted or not a directory: %v", p, err)
		}
		probe := filepath.Join(p, ".netductor-write-test")
		if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
			return p, fmt.Errorf("nfs path not writable: %w", err)
		}
		_ = os.Remove(probe)
		return p, nil
	case "local", "local_encrypted", "":
		if err := os.MkdirAll(p, 0o700); err != nil {
			return p, err
		}
		return p, nil
	default:
		return p, fmt.Errorf("unknown storage_backend %q", c.StorageBackend)
	}
}

// EnsureStorageReady prepares segment root according to backend.
func EnsureStorageReady() error {
	p, err := ResolveStoragePath()
	if err != nil {
		return err
	}
	c := LoadConfig()
	if c.Path != p {
		c.Path = p
		_ = SaveConfig(c)
	}
	return EnsureSegmentsDir()
}
