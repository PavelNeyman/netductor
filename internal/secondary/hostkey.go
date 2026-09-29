package secondary

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var pkhMu sync.Mutex

func relayKnownPath() string {
	return filepath.Join(paths.SecondaryDir(), "ssh_known_hosts.json")
}

// SSHHostEntry TOFU entry for secondary provision SSH.
type SSHHostEntry struct {
	ID        string `json:"id"`
	KeyPrefix string `json:"key_prefix"`
}

func loadRelayKH() map[string]string {
	f := map[string]string{}
	if b, err := os.ReadFile(relayKnownPath()); err == nil {
		_ = json.Unmarshal(b, &f)
	}
	if f == nil {
		f = map[string]string{}
	}
	return f
}

func saveRelayKH(f map[string]string) error {
	path := relayKnownPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("ssh known_hosts dir %s: %w", filepath.Dir(path), err)
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, 10), 0o600)
}

// ListSSHHosts lists secondary provision TOFU keys.
func ListSSHHosts() []SSHHostEntry {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	f := loadRelayKH()
	ids := make([]string, 0, len(f))
	for id := range f {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]SSHHostEntry, 0, len(ids))
	for _, id := range ids {
		k := f[id]
		pref := k
		if len(pref) > 16 {
			pref = pref[:16] + "…"
		}
		out = append(out, SSHHostEntry{ID: id, KeyPrefix: pref})
	}
	return out
}

// ForgetSSHHost removes one relay TOFU entry (no error if absent — safe for re-provision).
func ForgetSSHHost(id string) error {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	id = strings.TrimSpace(id)
	if h, _, err := net.SplitHostPort(id); err == nil {
		id = h
	}
	f := loadRelayKH()
	delete(f, id)
	return saveRelayKH(f)
}

// ClearSSHHosts removes all secondary provision TOFU keys.
func ClearSSHHosts() error {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	return saveRelayKH(map[string]string{})
}

func provisionHostKey(host string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		pkhMu.Lock()
		defer pkhMu.Unlock()
		f := loadRelayKH()
		got := base64.StdEncoding.EncodeToString(key.Marshal())
		id := host
		if prev, ok := f[id]; ok {
			if prev != got {
				return fmt.Errorf("SSH host key changed for %s — forget via: netductor ssh-hosts forget --kind relay %s", id, id)
			}
			return nil
		}
		f[id] = got
		return saveRelayKH(f)
	}
}
