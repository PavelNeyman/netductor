package edgestorage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)


// SafeDeviceID strips path separators and limits length for filesystem paths.
func SafeDeviceID(id string) string {
	id = strings.TrimSpace(id)
	id = filepath.Base(id) // drop any directory components
	id = strings.ReplaceAll(id, "..", "")
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

func Dir(deviceID string) string {
	id := SafeDeviceID(deviceID)
	if id == "" {
		id = "_invalid"
	}
	return filepath.Join("/var/lib/netductor/edge-backups", id)
}

type Manifest struct {
	DeviceID string `json:"device_id"`
	Kind     string `json:"kind"`
	Created  string `json:"created"`
	File     string `json:"file"`
	SHA256   string `json:"sha256,omitempty"`
	Note     string `json:"note,omitempty"`
}

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
		if e.IsDir() || !strings.HasPrefix(e.Name(), "manifest-") {
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
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out, nil
}

// LatestB returns newest type-B archive path for device.
func LatestB(deviceID string) (path string, name string, err error) {
	list, err := ListManifests(deviceID)
	if err != nil {
		return "", "", err
	}
	for _, m := range list {
		if m.Kind != "B" || m.File == "" {
			continue
		}
		p := filepath.Join(Dir(deviceID), filepath.Base(m.File))
		if st, e := os.Stat(p); e == nil && st.Size() > 0 {
			return p, m.File, nil
		}
	}
	// fallback: scan backup-B-*.tar.gz
	ents, err := os.ReadDir(Dir(deviceID))
	if err != nil {
		return "", "", fmt.Errorf("no type-B backup for %s", deviceID)
	}
	var names []string
	for _, e := range ents {
		n := e.Name()
		if strings.HasPrefix(n, "backup-B-") && strings.HasSuffix(n, ".tar.gz") {
			names = append(names, n)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) == 0 {
		return "", "", fmt.Errorf("no type-B backup for %s", deviceID)
	}
	return filepath.Join(Dir(deviceID), names[0]), names[0], nil
}

// OfflinePackManifest for USB offline restore.
type OfflinePackManifest struct {
	DeviceID string `json:"device_id"`
	Created  string `json:"created"`
	File     string `json:"file"`
	SHA256   string `json:"sha256"`
	Kind     string `json:"kind"`
}

// WriteOfflinePack copies latest B into destDir/nd-restore/ with manifest + sha.
func WriteOfflinePack(deviceID, destDir string) (string, error) {
	src, name, err := LatestB(deviceID)
	if err != nil {
		return "", err
	}
	destDir = clampOfflineDest(destDir, deviceID)
	outDir := filepath.Join(destDir, "nd-restore")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", err
	}
	dstName := "overlay-B.tar.gz"
	dst := filepath.Join(outDir, dstName)
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	w := io.MultiWriter(out, h)
	if _, err := io.Copy(w, in); err != nil {
		out.Close()
		return "", err
	}
	out.Close()
	sum := hex.EncodeToString(h.Sum(nil))
	man := OfflinePackManifest{
		DeviceID: deviceID,
		Created:  time.Now().UTC().Format(time.RFC3339),
		File:     dstName,
		SHA256:   sum,
		Kind:     "B",
		}
	_ = name
	b, _ := json.MarshalIndent(man, "", "  ")
	_ = os.WriteFile(filepath.Join(outDir, "manifest.json"), b, 0o600)
	_ = os.WriteFile(filepath.Join(outDir, "SHA256SUMS"), []byte(sum+"  "+dstName+"\n"), 0o600)
	return outDir, nil
}


// clampOfflineDest forces packs under /var/lib/netductor/edge-offline/<safe-id>/…
func clampOfflineDest(dest, deviceID string) string {
	base := "/var/lib/netductor/edge-offline"
	safe := SafeDeviceID(deviceID)
	if safe == "" {
		safe = "_unknown"
	}
	defaultDir := filepath.Join(base, safe)
	if dest == "" {
		return defaultDir
	}
	clean := filepath.Clean(dest)
	// must stay under base
	if clean == base || strings.HasPrefix(clean, base+string(filepath.Separator)) {
		return clean
	}
	return defaultDir
}
