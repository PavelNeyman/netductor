package inventory

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const (
	DefaultDir  = "/etc/netductor/inventory"
	DefaultFile = DefaultDir + "/secondaries.json"
)

var idRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var mu sync.Mutex

// Secondary is a fleet candidate / inventory entry (not the same as service-plane peer).
type Secondary struct {
	ID            string    `json:"id"`
	Name          string    `json:"name,omitempty"`
	PublicIP      string    `json:"public_ip"`
	Lifecycle     string    `json:"lifecycle_state"` // planned|provisioning|ready|active|degraded|removing|removed
	DeployState   string    `json:"deploy_state,omitempty"`
	ServicePeerID string    `json:"service_peer_id,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type File struct {
	Version int         `json:"version"`
	Items   []Secondary `json:"items"`
}

func load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: 1}, nil
		}
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("inventory corrupt: %w", err)
	}
	if f.Version == 0 {
		f.Version = 1
	}
	return &f, nil
}

func save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// List returns all non-removed entries.
func List(path string) ([]Secondary, error) {
	mu.Lock()
	defer mu.Unlock()
	f, err := load(path)
	if err != nil {
		return nil, err
	}
	var out []Secondary
	for _, s := range f.Items {
		if s.Lifecycle != "removed" {
			out = append(out, s)
		}
	}
	return out, nil
}

// Register creates a planned candidate (no network changes).
func Register(path, id, name, publicIP, notes string) (*Secondary, error) {
	mu.Lock()
	defer mu.Unlock()
	if id == "" {
		id = "sec-" + regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(publicIP, "-")
	}
	if !idRe.MatchString(id) {
		return nil, fmt.Errorf("invalid id %q", id)
	}
	if net.ParseIP(publicIP) == nil {
		return nil, fmt.Errorf("invalid public_ip %q", publicIP)
	}
	f, err := load(path)
	if err != nil {
		return nil, err
	}
	for _, s := range f.Items {
		if s.ID == id && s.Lifecycle != "removed" {
			return nil, fmt.Errorf("id %s already exists", id)
		}
		if s.PublicIP == publicIP && s.Lifecycle != "removed" {
			return nil, fmt.Errorf("public_ip %s already registered as %s", publicIP, s.ID)
		}
	}
	now := time.Now().UTC()
	s := Secondary{
		ID: id, Name: name, PublicIP: publicIP, Lifecycle: "planned",
		Notes: notes, CreatedAt: now, UpdatedAt: now,
	}
	f.Items = append(f.Items, s)
	if err := save(path, f); err != nil {
		return nil, err
	}
	return &s, nil
}

// Get returns one entry by id.
func Get(path, id string) (*Secondary, error) {
	mu.Lock()
	defer mu.Unlock()
	f, err := load(path)
	if err != nil {
		return nil, err
	}
	for i := range f.Items {
		if f.Items[i].ID == id {
			s := f.Items[i]
			return &s, nil
		}
	}
	return nil, fmt.Errorf("not found: %s", id)
}

// UpdateLifecycle sets lifecycle / service_peer link.
func UpdateLifecycle(path, id, lifecycle, peerID, deployState string) error {
	mu.Lock()
	defer mu.Unlock()
	f, err := load(path)
	if err != nil {
		return err
	}
	for i := range f.Items {
		if f.Items[i].ID == id {
			if lifecycle != "" {
				f.Items[i].Lifecycle = lifecycle
			}
			if peerID != "" {
				f.Items[i].ServicePeerID = peerID
			}
			if deployState != "" {
				f.Items[i].DeployState = deployState
			}
			f.Items[i].UpdatedAt = time.Now().UTC()
			return save(path, f)
		}
	}
	return fmt.Errorf("not found: %s", id)
}

// SoftRemove marks removed.
func SoftRemove(path, id string) error {
	return UpdateLifecycle(path, id, "removed", "", "")
}
