// Package nvr — camera inventory and NVR metadata (Phase A+).
package nvr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var mu sync.Mutex

// Camera is a multi-site camera record (secrets by reference, not in git).
type Camera struct {
	ID           string            `json:"id"`
	SiteID       string            `json:"site_id"`       // edge device_id or location id
	Name         string            `json:"name"`
	MAC          string            `json:"mac,omitempty"`
	LANIP        string            `json:"lan_ip,omitempty"`
	RTSPUser     string            `json:"rtsp_user,omitempty"`
	RTSPPath     string            `json:"rtsp_path"` // e.g. /stream1
	RTSPPort     int               `json:"rtsp_port"`
	SecretRef    string            `json:"secret_ref,omitempty"` // key in secrets file
	Features     map[string]bool   `json:"features,omitempty"`  // ptz, night, onvif
	Enabled      bool              `json:"enabled"`
	Record       bool              `json:"record"`
	StreamSub    bool              `json:"stream_sub"` // prefer stream2 for motion
	Created      int64             `json:"created"`
	Updated      int64             `json:"updated"`
	Meta         map[string]string `json:"meta,omitempty"`
}

type storeFile struct {
	Cameras []Camera `json:"cameras"`
}

func dir() string {
	return filepath.Join(paths.StateDir(), "nvr")
}

func camerasPath() string {
	return filepath.Join(dir(), "cameras.json")
}

func secretsPath() string {
	return filepath.Join(dir(), "secrets.json")
}

func ensure() error {
	return os.MkdirAll(dir(), 0o700)
}

func load() storeFile {
	var s storeFile
	b, err := os.ReadFile(camerasPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func save(s storeFile) error {
	if err := ensure(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := camerasPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, camerasPath())
}

// ListCameras returns all cameras.
func ListCameras() []Camera {
	mu.Lock()
	defer mu.Unlock()
	return append([]Camera(nil), load().Cameras...)
}

// GetCamera by id.
func GetCamera(id string) (Camera, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, c := range load().Cameras {
		if c.ID == id {
			return c, true
		}
	}
	return Camera{}, false
}

// UpsertCamera creates or updates by id (empty id → new).
func UpsertCamera(c Camera) (Camera, error) {
	mu.Lock()
	defer mu.Unlock()
	_ = ensure()
	s := load()
	now := time.Now().Unix()
	if c.RTSPPath == "" {
		c.RTSPPath = "/stream1"
	}
	if c.RTSPPort == 0 {
		c.RTSPPort = 554
	}
	if c.ID == "" {
		c.ID = randomID()
		c.Created = now
	}
	c.Updated = now
	c.MAC = normalizeMAC(c.MAC)
	found := false
	for i := range s.Cameras {
		if s.Cameras[i].ID == c.ID {
			if c.Created == 0 {
				c.Created = s.Cameras[i].Created
			}
			s.Cameras[i] = c
			found = true
			break
		}
	}
	if !found {
		if c.Created == 0 {
			c.Created = now
		}
		s.Cameras = append(s.Cameras, c)
	}
	if err := save(s); err != nil {
		return Camera{}, err
	}
	return c, nil
}

// DeleteCamera removes by id.
func DeleteCamera(id string) bool {
	mu.Lock()
	defer mu.Unlock()
	s := load()
	out := s.Cameras[:0]
	ok := false
	for _, c := range s.Cameras {
		if c.ID == id {
			ok = true
			continue
		}
		out = append(out, c)
	}
	s.Cameras = out
	_ = save(s)
	return ok
}

// SetSecret stores RTSP password under ref (camera id by default).
func SetSecret(ref, password string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := ensure(); err != nil {
		return err
	}
	m := map[string]string{}
	if b, err := os.ReadFile(secretsPath()); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	m[ref] = password
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := secretsPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, secretsPath())
}

// GetSecret returns password for ref.
func GetSecret(ref string) string {
	mu.Lock()
	defer mu.Unlock()
	b, err := os.ReadFile(secretsPath())
	if err != nil {
		return ""
	}
	m := map[string]string{}
	_ = json.Unmarshal(b, &m)
	return m[ref]
}

// RTSPURL builds rtsp URL if secret present.
func RTSPURL(c Camera) string {
	pass := GetSecret(c.SecretRef)
	if pass == "" && c.SecretRef == "" {
		pass = GetSecret(c.ID)
	}
	user := c.RTSPUser
	if user == "" {
		user = "cam"
	}
	host := c.LANIP
	if host == "" {
		return ""
	}
	path := c.RTSPPath
	if path == "" {
		path = "/stream1"
	}
	if c.StreamSub && (path == "/stream1" || path == "stream1") {
		path = "/stream2"
	}
	port := c.RTSPPort
	if port == 0 {
		port = 554
	}
	if pass == "" {
		return ""
	}
	return "rtsp://" + user + ":" + pass + "@" + host + ":" + itoa(port) + path
}

func normalizeMAC(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", ":")
	return s
}

func randomID() string {
	return strings.ReplaceAll(time.Now().Format("20060102150405.000"), ".", "") + randSuffix()
}

func randSuffix() string {
	b := make([]byte, 3)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "x"
	}
	defer f.Close()
	_, _ = f.Read(b)
	const hex = "0123456789abcdef"
	out := make([]byte, 6)
	for i := 0; i < 3; i++ {
		out[i*2] = hex[b[i]>>4]
		out[i*2+1] = hex[b[i]&0xf]
	}
	return string(out)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [16]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}
