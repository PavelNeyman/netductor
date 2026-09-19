package nodes

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Node is a managed host in the fleet (VPS, OpenWrt, MikroTik, …).
type Node struct {
	ID        string            `json:"id"`
	Hostname  string            `json:"hostname"`
	Role      string            `json:"role"` // core | edge | lab
	Kind      string            `json:"kind"` // vps | openwrt | mikrotik
	PublicIP  string            `json:"public_ip,omitempty"`
	Status    string            `json:"status"` // online | offline | pending
	LastSeen  int64             `json:"last_seen,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	DesiredHN string            `json:"desired_hostname,omitempty"` // operator rename pending on device
	Updated   int64             `json:"updated"`
}

type fileData struct {
	Nodes map[string]Node `json:"nodes"`
}

var mu sync.Mutex

func path() string {
	return filepath.Join(paths.StateDir(), "nodes", "registry.json")
}

func load() (fileData, error) {
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	b, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return fileData{Nodes: map[string]Node{}}, nil
		}
		return fileData{}, err
	}
	var d fileData
	if err := json.Unmarshal(b, &d); err != nil {
		return fileData{}, err
	}
	if d.Nodes == nil {
		d.Nodes = map[string]Node{}
	}
	return d, nil
}

func save(d fileData) error {
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := path() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path())
}

func NormalizeHostname(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

// Upsert from device heartbeat / install (device is source of truth for current hostname/ip).
func UpsertFromDevice(n Node) (Node, error) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return Node{}, err
	}
	n.ID = strings.TrimSpace(n.ID)
	if n.ID == "" {
		return Node{}, fmt.Errorf("id required")
	}
	n.Hostname = NormalizeHostname(n.Hostname)
	now := time.Now().Unix()
	n.Updated = now
	if n.LastSeen == 0 {
		n.LastSeen = now
	}
	if n.Status == "" {
		n.Status = "online"
	}
	if prev, ok := d.Nodes[n.ID]; ok {
		// keep desired hostname set by operator if not yet applied
		if n.Hostname != "" && n.Hostname == prev.DesiredHN {
			n.DesiredHN = ""
		} else if prev.DesiredHN != "" && n.Hostname != prev.DesiredHN {
			n.DesiredHN = prev.DesiredHN
		}
		if n.Role == "" {
			n.Role = prev.Role
		}
		if n.Kind == "" {
			n.Kind = prev.Kind
		}
		if n.Labels == nil {
			n.Labels = prev.Labels
		}
	}
	d.Nodes[n.ID] = n
	if err := save(d); err != nil {
		return Node{}, err
	}
	return n, nil
}

// SetDesiredHostname: operator changes name in registry → device should apply on next poll.
func SetDesiredHostname(id, hostname string) (Node, error) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return Node{}, err
	}
	n, ok := d.Nodes[id]
	if !ok {
		return Node{}, fmt.Errorf("unknown node %s", id)
	}
	hostname = NormalizeHostname(hostname)
	if hostname == "" {
		return Node{}, fmt.Errorf("empty hostname")
	}
	n.DesiredHN = hostname
	n.Updated = time.Now().Unix()
	d.Nodes[id] = n
	if err := save(d); err != nil {
		return Node{}, err
	}
	return n, nil
}

func List() ([]Node, error) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return nil, err
	}
	out := make([]Node, 0, len(d.Nodes))
	for _, n := range d.Nodes {
		out = append(out, n)
	}
	return out, nil
}

func Get(id string) (Node, bool, error) {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return Node{}, false, err
	}
	n, ok := d.Nodes[id]
	return n, ok, nil
}

func Delete(id string) error {
	mu.Lock()
	defer mu.Unlock()
	d, err := load()
	if err != nil {
		return err
	}
	delete(d.Nodes, id)
	return save(d)
}

// SelfRegisterLocal VPS after install/rename. ID is stable UUID; hostname is separate.
func SelfRegisterLocal(hostname, role, publicIP string) error {
	id := LocalStableID()
	hostname = NormalizeHostname(hostname)
	if hostname == "" {
		hostname = LocalHostname()
	}
	if role == "" {
		role = "core"
	}
	_, err := UpsertFromDevice(Node{
		ID:       id,
		Hostname: hostname,
		Role:     role,
		Kind:     "vps",
		PublicIP: publicIP,
		Status:   "online",
	})
	return err
}

// SyncLocalHostname applies desired_hostname for this host if set in registry.
// Stable ID never changes on rename.
func SyncLocalHostname() error {
	id := LocalStableID()
	hn := LocalHostname()
	ip := ""
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_ip")); err == nil {
		ip = strings.TrimSpace(string(b))
	}

	list, err := List()
	if err != nil {
		return err
	}

	var mine *Node
	for i := range list {
		n := list[i]
		if n.ID == id || (n.Kind == "vps" && n.Hostname == hn && hn != "") {
			mine = &list[i]
			break
		}
	}
	// migrate legacy entry where id == old hostname
	if mine == nil {
		for i := range list {
			if list[i].Kind == "vps" && list[i].PublicIP == ip && ip != "" {
				mine = &list[i]
				break
			}
		}
	}
	if mine == nil {
		return SelfRegisterLocal(hn, "core", ip)
	}

	role := mine.Role
	if role == "" {
		role = "core"
	}
	want := mine.DesiredHN
	oldID := mine.ID

	if want != "" && want != mine.Hostname {
		WriteLocalHostname(want)
		_ = exec.Command("hostnamectl", "set-hostname", want).Run()
		hn = want
	} else if hn == "" {
		hn = mine.Hostname
	}

	_, err = UpsertFromDevice(Node{
		ID: id, Hostname: hn, Role: role, Kind: "vps", PublicIP: ip, Status: "online",
	})
	if err != nil {
		return err
	}
	// drop legacy id==hostname entries and other same-IP vps rows
	if oldID != id {
		_ = Delete(oldID)
	}
	_ = pruneStaleVPS(id, ip)
	return nil
}

// pruneStaleVPS removes other vps entries with same public IP but different id.
func pruneStaleVPS(keepID, publicIP string) error {
	if publicIP == "" {
		return nil
	}
	list, err := List()
	if err != nil {
		return err
	}
	for _, n := range list {
		if n.Kind == "vps" && n.PublicIP == publicIP && n.ID != keepID {
			_ = Delete(n.ID)
		}
	}
	return nil
}




// MarkStaleRelays sets offline for relay nodes not seen recently and drops empty-IP ghosts.
func MarkStaleRelays(maxAgeSec int64) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return
	}
	if r.Nodes == nil {
		r.Nodes = map[string]Node{}
	}
	now := time.Now().Unix()
	changed := false
	for id, n := range r.Nodes {
		if n.Role != "secondary" {
			continue
		}
		if n.PublicIP == "" && (n.LastSeen == 0 || now-n.LastSeen > maxAgeSec) {
			delete(r.Nodes, id)
			changed = true
			continue
		}
		if n.LastSeen > 0 && now-n.LastSeen > maxAgeSec && n.Status != "offline" {
			n.Status = "offline"
			r.Nodes[id] = n
			changed = true
		}
	}
	if changed {
		_ = save(r)
	}
}
