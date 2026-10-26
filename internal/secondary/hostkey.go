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

// loadRelayKH returns TOFU map. Missing file → empty map.
// Corrupt/unreadable file → error (never treat as empty — would allow re-pin MITM).
func loadRelayKH() (map[string]string, error) {
	b, err := os.ReadFile(relayKnownPath())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("ssh TOFU read: %w", err)
	}
	f := map[string]string{}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("ssh TOFU corrupt (fix or: netductor ssh-hosts clear): %w", err)
	}
	if f == nil {
		f = map[string]string{}
	}
	return f, nil
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
	f, err := loadRelayKH()
	if err != nil {
		return nil
	}
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
	f, err := loadRelayKH()
	if err != nil {
		return err
	}
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
		f, err := loadRelayKH()
		if err != nil {
			return err
		}
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

// OpenSSHKnownHostsLine builds a StrictHostKeyChecking=yes line from TOFU store.
// Empty string if host not yet pinned (first contact still uses accept-new only when no pin).
func OpenSSHKnownHostsLine(host, port string) (string, error) {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	f, err := loadRelayKH()
	if err != nil {
		return "", err
	}
	b64, ok := f[host]
	if !ok || b64 == "" {
		return "", fmt.Errorf("no TOFU pin for %s — run provision SSH first", host)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("TOFU key corrupt for %s", host)
	}
	pk, err := ssh.ParsePublicKey(raw)
	if err != nil {
		return "", fmt.Errorf("TOFU key parse for %s: %w", host, err)
	}
	ak := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk)))
	// ak is "type base64"
	name := host
	if port != "" && port != "22" {
		name = fmt.Sprintf("[%s]:%s", host, port)
	}
	return name + " " + ak, nil
}

// WriteOpenSSHKnownHostsFile writes a 0600 known_hosts file for OpenSSH strict checking.
func WriteOpenSSHKnownHostsFile(host, port, dest string) error {
	line, err := OpenSSHKnownHostsLine(host, port)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	return os.WriteFile(dest, append([]byte(line), 10), 0o600)
}
