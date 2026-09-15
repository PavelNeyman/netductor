package sites

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

// Site groups MikroTik (routing) + RPi OpenWrt (VPN edge) as one logical location.
type Site struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RPiID      string `json:"rpi_id,omitempty"`
	MikroTikID string `json:"mikrotik_id,omitempty"`
	Notes      string `json:"notes,omitempty"`
	Updated    int64  `json:"updated"`
}

type fileData struct {
	Sites map[string]Site `json:"sites"`
}

var mu sync.Mutex

func path() string {
	return filepath.Join(paths.StateDir(), "sites", "sites.json")
}

func load() (fileData, error) {
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	b, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return fileData{Sites: map[string]Site{}}, nil
		}
		return fileData{}, err
	}
	var d fileData
	if err := json.Unmarshal(b, &d); err != nil {
		return fileData{}, err
	}
	if d.Sites == nil {
		d.Sites = map[string]Site{}
	}
	return d, nil
}

func save(d fileData) error {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path(), append(raw, '\n'), 0o600)
}

func List() ([]Site, error) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return nil, err
	}
	out := make([]Site, 0, len(d.Sites))
	for _, s := range d.Sites {
		out = append(out, s)
	}
	return out, nil
}

func Upsert(s Site) (Site, error) {
	if s.ID == "" {
		return s, fmt.Errorf("id required")
	}
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return s, err
	}
	s.Updated = time.Now().Unix()
	d.Sites[s.ID] = s
	return s, save(d)
}

func Get(id string) (Site, bool) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return Site{}, false
	}
	s, ok := d.Sites[id]
	return s, ok
}

// RSCForSite generates MikroTik routing-only script for the site.
func RSCForSite(id string) (string, error) {
	return RSCForSiteWithGateway(id, "")
}

func RSCForSiteWithGateway(id, rpiLAN string) (string, error) {
	s, ok := Get(id)
	if !ok {
		s = Site{ID: id, Name: id}
	}
	name := s.MikroTikID
	if name == "" {
		name = "mt-" + s.ID
	}
	relayIP := ""
	for _, d := range secondary.List() {
		if d.PublicIP != "" {
			relayIP = d.PublicIP
			break
		}
	}
	note := s.Name
	if note == "" {
		note = "netductor site " + s.ID
	}
	base := mikrotik.ClientRSC(name, relayIP, "", note)
	if rpiLAN == "" {
		return base, nil
	}
	nl := string([]byte{10})
	extra := nl + "# netductor routes via RPi" + nl
	extra += "/routing table add fib name=via-rpi" + nl
	extra += "/ip route add dst-address=0.0.0.0/0 gateway=" + rpiLAN + " routing-table=via-rpi" + nl
	return base + extra, nil
}
