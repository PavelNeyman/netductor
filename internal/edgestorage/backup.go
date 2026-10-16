package edgestorage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Dir for edge device backups on primary.
func Dir(deviceID string) string {
	return filepath.Join("/var/lib/netductor/edge-backups", deviceID)
}

// Manifest describes a stored backup slot.
type Manifest struct {
	DeviceID  string   `json:"device_id"`
	Kind      string   `json:"kind"` // A | B
	Created   string   `json:"created"`
	File      string   `json:"file"`
	SHA256    string   `json:"sha256,omitempty"`
	OpenWrt   string   `json:"openwrt,omitempty"`
	Note      string   `json:"note,omitempty"`
}

// SaveManifest writes manifest next to archive.
func SaveManifest(deviceID string, m Manifest) error {
	dir := Dir(deviceID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	m.DeviceID = deviceID
	if m.Created == "" {
		m.Created = time.Now().UTC().Format(time.RFC3339)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	name := fmt.Sprintf("manifest-%s-%s.json", m.Kind, time.Now().UTC().Format("20060102T150405Z"))
	return os.WriteFile(filepath.Join(dir, name), b, 0o600)
}

// ListManifests returns manifests for device (best-effort).
func ListManifests(deviceID string) ([]Manifest, error) {
	dir := Dir(deviceID)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Manifest
	for _, e := range ents {
		if e.IsDir() || !stringsHasPrefix(e.Name(), "manifest-") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var m Manifest
		if json.Unmarshal(b, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
