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

func Dir(deviceID string) string {
	return filepath.Join("/var/lib/netductor/edge-backups", deviceID)
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
