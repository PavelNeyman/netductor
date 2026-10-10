package devices

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Device is a client endpoint observed on the VPN (bound to a vpn user name).
type Device struct {
	ID         string    `json:"id"` // user|srcIP
	User       string    `json:"user"`
	SrcIP      string    `json:"src_ip"`
	LastSeen   time.Time `json:"last_seen"`
	LastDest   string    `json:"last_dest,omitempty"`
	Hits       int       `json:"hits"`
	Inbound    string    `json:"inbound,omitempty"` // vless-reality | relay-in | …
	Note       string    `json:"note,omitempty"`
}

type fileStore struct {
	Devices []Device `json:"devices"`
}

var mu sync.Mutex

func storePath() string {
	return filepath.Join(paths.StateDir(), "vpn-devices.json")
}

var (
	// [Pavel] inbound connection from 46.x:port
	reUserFrom = regexp.MustCompile(`\[([A-Za-z0-9._-]+)\][^\n]*inbound connection from ([0-9.]+):`)
	// inbound connection from IP without name — skip or unknown
	reFrom = regexp.MustCompile(`inbound connection from ([0-9.]+):`)
	reDest = regexp.MustCompile(`inbound connection to ([^\s]+)`)
)

// RefreshFromJournal scans recent sing-box logs and merges into store.
func RefreshFromJournal(windowMin int) ([]Device, error) {
	if windowMin < 1 {
		windowMin = 60
	}
	out, err := exec.Command("journalctl", "-u", "sing-box",
		"--since", time.Now().Add(-time.Duration(windowMin)*time.Minute).Format("2006-01-02 15:04:05"),
		"-o", "cat", "--no-pager").CombinedOutput()
	if err != nil {
		// still return store
		return List(), nil
	}
	mu.Lock()
	defer mu.Unlock()
	cur, _ := loadUnlocked()
	byID := map[string]int{}
	for i, d := range cur {
		byID[d.ID] = i
	}
	now := time.Now().UTC()
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "inbound connection") {
			continue
		}
		user, ip := "", ""
		if m := reUserFrom.FindStringSubmatch(line); len(m) == 3 {
			user, ip = m[1], m[2]
		} else {
			continue // need named user for locator binding
		}
		if user == "" || user == "relay-uplink" {
			continue
		}
		dest := ""
		if m := reDest.FindStringSubmatch(line); len(m) == 2 {
			dest = m[1]
		}
		id := user + "|" + ip
		if i, ok := byID[id]; ok {
			cur[i].LastSeen = now
			cur[i].Hits++
			if dest != "" {
				cur[i].LastDest = dest
			}
		} else {
			d := Device{ID: id, User: user, SrcIP: ip, LastSeen: now, LastDest: dest, Hits: 1}
			byID[id] = len(cur)
			cur = append(cur, d)
		}
	}
	// prune > 30d and cap entries
	cut := now.Add(-30 * 24 * time.Hour)
	kept := cur[:0]
	for _, d := range cur {
		if d.LastSeen.After(cut) {
			kept = append(kept, d)
		}
	}
	const maxDeviceEntries = 500
	if len(kept) > maxDeviceEntries {
		sort.Slice(kept, func(i, j int) bool { return kept[i].LastSeen.After(kept[j].LastSeen) })
		kept = kept[:maxDeviceEntries]
	}
	_ = saveUnlocked(kept)
	return kept, nil
}

func loadUnlocked() ([]Device, error) {
	b, err := os.ReadFile(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f fileStore
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.Devices, nil
}

func saveUnlocked(list []Device) error {
	_ = os.MkdirAll(filepath.Dir(storePath()), 0o700)
	b, _ := json.MarshalIndent(fileStore{Devices: list}, "", "  ")
	return os.WriteFile(storePath(), append(b, '\n'), 0o600)
}

// List returns known devices (newest first).
func List() []Device {
	mu.Lock()
	defer mu.Unlock()
	list, _ := loadUnlocked()
	sort.Slice(list, func(i, j int) bool { return list[i].LastSeen.After(list[j].LastSeen) })
	return list
}

// ListByUser filters.
func ListByUser(user string) []Device {
	user = strings.TrimSpace(user)
	var out []Device
	for _, d := range List() {
		if d.User == user {
			out = append(out, d)
		}
	}
	return out
}
