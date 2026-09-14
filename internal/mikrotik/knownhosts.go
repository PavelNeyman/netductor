package mikrotik

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var khMu sync.Mutex

func knownHostsPath() string {
	return filepath.Join(paths.StateDir(), "mikrotik", "known_hosts.json")
}

type khFile struct {
	Hosts map[string]string `json:"hosts"` // host:port -> base64 raw key
}

// HostEntry is a stored TOFU host key.
type HostEntry struct {
	ID        string `json:"id"` // host:port
	KeyPrefix string `json:"key_prefix"`
}

func loadKH() khFile {
	b, err := os.ReadFile(knownHostsPath())
	if err != nil {
		return khFile{Hosts: map[string]string{}}
	}
	var f khFile
	if json.Unmarshal(b, &f) != nil || f.Hosts == nil {
		return khFile{Hosts: map[string]string{}}
	}
	return f
}

func saveKH(f khFile) error {
	_ = os.MkdirAll(filepath.Dir(knownHostsPath()), 0o700)
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(knownHostsPath(), append(raw, 10), 0o600)
}

func hostKeyKey(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

// ListKnownHosts returns all TOFU entries for MikroTik SSH.
func ListKnownHosts() []HostEntry {
	khMu.Lock()
	defer khMu.Unlock()
	f := loadKH()
	ids := make([]string, 0, len(f.Hosts))
	for id := range f.Hosts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]HostEntry, 0, len(ids))
	for _, id := range ids {
		k := f.Hosts[id]
		pref := k
		if len(pref) > 16 {
			pref = pref[:16] + "…"
		}
		out = append(out, HostEntry{ID: id, KeyPrefix: pref})
	}
	return out
}

// ForgetKnownHost removes one entry (id = host:port or host).
func ForgetKnownHost(id string) error {
	khMu.Lock()
	defer khMu.Unlock()
	f := loadKH()
	if _, ok := f.Hosts[id]; ok {
		delete(f.Hosts, id)
		return saveKH(f)
	}
	// try host without port variants
	deleted := false
	for k := range f.Hosts {
		if k == id || len(k) > len(id) && k[:len(id)] == id && k[len(id)] == ':' {
			delete(f.Hosts, k)
			deleted = true
		}
	}
	if !deleted {
		return fmt.Errorf("not found: %s", id)
	}
	return saveKH(f)
}

// ClearKnownHosts removes all MikroTik TOFU entries.
func ClearKnownHosts() error {
	khMu.Lock()
	defer khMu.Unlock()
	return saveKH(khFile{Hosts: map[string]string{}})
}

// hostKeyCallback TOFU: first connect stores key; later verifies.
// NETDUCTOR_MT_STRICT=1 rejects unknown hosts instead of storing.
func hostKeyCallback(host string, port int) ssh.HostKeyCallback {
	strict := os.Getenv("NETDUCTOR_MT_STRICT") == "1"
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		khMu.Lock()
		defer khMu.Unlock()
		f := loadKH()
		id := hostKeyKey(host, port)
		got := base64.StdEncoding.EncodeToString(key.Marshal())
		if prev, ok := f.Hosts[id]; ok {
			if prev != got {
				return fmt.Errorf("host key mismatch for %s (TOFU) — forget via: netductor ssh-hosts forget %s", id, id)
			}
			return nil
		}
		if strict {
			return fmt.Errorf("unknown host key for %s (strict mode)", id)
		}
		f.Hosts[id] = got
		return saveKH(f)
	}
}
