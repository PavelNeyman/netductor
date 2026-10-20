package sites

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/statefile"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

// Site groups MikroTik (routing) + RPi OpenWrt (VPN edge) as one logical location.
// Location (stored as Site) is a logical place (home, flat, office): a group of edge devices
// (OpenWrt agents) and optional MikroTik. Not a single router — a set.
// Location is the product name for Site (home / flat / room inventory).
type Location = Site

type Site struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`               // e.g. "home", "office-msk"
	Kind       string   `json:"kind,omitempty"`     // home|flat|office|other
	EdgeIDs    []string `json:"edge_ids,omitempty"` // netductor-agent device_ids
	RPiID      string   `json:"rpi_id,omitempty"`
	MikroTikID string   `json:"mikrotik_id,omitempty"`
	Notes      string   `json:"notes,omitempty"`
	Updated    int64    `json:"updated"`
}

type fileData struct {
	Sites map[string]Site `json:"sites"`
}

var mu sync.Mutex

func path() string {
	return filepath.Join(paths.StateDir(), "sites", "sites.json")
}

func load() (fileData, error) {
	var d fileData
	err := statefile.LoadJSON(path(), &d)
	if statefile.IsNotExist(err) {
		return fileData{Sites: map[string]Site{}}, nil
	}
	if err != nil {
		return fileData{}, err
	}
	if d.Sites == nil {
		d.Sites = map[string]Site{}
	}
	return d, nil
}

func save(d fileData) error {
	return statefile.SaveJSON(path(), d)
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

func Delete(id string) error {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return err
	}
	delete(d.Sites, id)
	return save(d)
}

func Get(id string) (Site, bool) {
	s, ok, err := GetStrict(id)
	if err != nil {
		return Site{}, false
	}
	return s, ok
}

// GetStrict distinguishes missing site (ok=false, err=nil) from store failure (err!=nil).
func GetStrict(id string) (Site, bool, error) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return Site{}, false, err
	}
	s, ok := d.Sites[id]
	return s, ok, nil
}

// RSCForSite generates MikroTik routing-only script for the site.
func RSCForSite(id string) (string, error) {
	return RSCForSiteWithGateway(id, "")
}

func RSCForSiteWithGateway(id, rpiLAN string) (string, error) {
	s, ok, err := GetStrict(id)
	if err != nil {
		return "", err
	}
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

// AttachEdge adds device_id to site EdgeIDs (dedup).
func AttachEdge(siteID, deviceID string) error {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return err
	}
	s, ok := d.Sites[siteID]
	if !ok {
		return fmt.Errorf("unknown site %s", siteID)
	}
	for _, id := range s.EdgeIDs {
		if id == deviceID {
			s.Updated = time.Now().Unix()
			d.Sites[siteID] = s
			return save(d)
		}
	}
	s.EdgeIDs = append(s.EdgeIDs, deviceID)
	s.Updated = time.Now().Unix()
	d.Sites[siteID] = s
	return save(d)
}
