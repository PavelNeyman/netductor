package nvr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Config is global NVR settings (storage + retention + segment length).
type Config struct {
	// StorageBackend: local_encrypted | local | nfs | home_nfs
	StorageBackend string `json:"storage_backend"`
	// Path is root for segments (must be on encrypted mount when required).
	Path string `json:"path"`
	// SegmentSec length of each file (default 300).
	SegmentSec int `json:"segment_sec"`
	// RetentionDays delete segments older than N days (0 = ignore age).
	RetentionDays int `json:"retention_days"`
	// MaxGB total size cap across all cameras (0 = ignore size).
	MaxGB float64 `json:"max_gb"`
	// MinFreeGB stop writing / force rotate if free space below (0 = off).
	MinFreeGB float64 `json:"min_free_gb"`
	// RotateIntervalSec how often retention runs (default 300).
	RotateIntervalSec int `json:"rotate_interval_sec"`
	// RecordEnabled global kill switch.
	RecordEnabled bool `json:"record_enabled"`
}

func defaultConfig() Config {
	return Config{
		StorageBackend:    "local_encrypted",
		Path:              filepath.Join(paths.StateDir(), "nvr", "segments"),
		SegmentSec:        300,
		RetentionDays:     7,
		MaxGB:             40,
		MinFreeGB:         5,
		RotateIntervalSec: 300,
		RecordEnabled:     false, // off until operator enables + path ready
	}
}

func configPath() string {
	return filepath.Join(dir(), "config.json")
}

// LoadConfig returns NVR config (defaults merged).
func LoadConfig() Config {
	mu.Lock()
	defer mu.Unlock()
	return loadConfigUnlocked()
}

func loadConfigUnlocked() Config {
	c := defaultConfig()
	b, err := os.ReadFile(configPath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	if c.Path == "" {
		c.Path = defaultConfig().Path
	}
	if c.SegmentSec <= 0 {
		c.SegmentSec = 300
	}
	if c.RotateIntervalSec <= 0 {
		c.RotateIntervalSec = 300
	}
	if c.StorageBackend == "" {
		c.StorageBackend = "local_encrypted"
	}
	return c
}

// SaveConfig persists NVR config.
func SaveConfig(c Config) error {
	mu.Lock()
	defer mu.Unlock()
	if err := ensure(); err != nil {
		return err
	}
	def := defaultConfig()
	if c.Path == "" {
		c.Path = def.Path
	}
	if c.SegmentSec <= 0 {
		c.SegmentSec = def.SegmentSec
	}
	if c.RotateIntervalSec <= 0 {
		c.RotateIntervalSec = def.RotateIntervalSec
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}

// SegmentsRoot returns configured path (best-effort resolve).
func SegmentsRoot() string {
	p, err := ResolveStoragePath()
	if err != nil {
		return LoadConfig().Path
	}
	return p
}

// EnsureSegmentsDir creates segment root.
func EnsureSegmentsDir() error {
	c := LoadConfig()
	return os.MkdirAll(c.Path, 0o700)
}

// Now is overridable in tests.
var Now = time.Now
