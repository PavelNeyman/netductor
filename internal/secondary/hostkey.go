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

func relaySeenPath() string {
	return filepath.Join(paths.SecondaryDir(), "ssh_hosts_seen.json")
}

// ErrPinLost: host was pinned before but entry is gone — not first contact.
var ErrPinLost = fmt.Errorf("TOFU pin lost for previously known host")

func loadSeenHosts() (map[string]bool, error) {
	b, err := os.ReadFile(relaySeenPath())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("ssh seen-hosts read: %w", err)
	}
	var ids []string
	if err := json.Unmarshal(b, &ids); err != nil {
		return nil, fmt.Errorf("ssh seen-hosts corrupt: %w", err)
	}
	out := map[string]bool{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out[id] = true
		}
	}
	return out, nil
}

func saveSeenHosts(m map[string]bool) error {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	path := relaySeenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, append(raw, 10), 0o600)
}

// ensureSeenMigratedFromPins adds every current pin host to seen (idempotent).
// Call under pkhMu. Fixes upgrades from pre-0.9.574 installs that only had ssh_known_hosts.json.
func ensureSeenMigratedFromPins(pins map[string]string) error {
	if len(pins) == 0 {
		return nil
	}
	seen, err := loadSeenHosts()
	if err != nil {
		return err
	}
	changed := false
	for id := range pins {
		if id == "" {
			continue
		}
		if !seen[id] {
			seen[id] = true
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return saveSeenHosts(seen)
}

func markHostSeen(id string) error {
	m, err := loadSeenHosts()
	if err != nil {
		return err
	}
	m[id] = true
	return saveSeenHosts(m)
}

func unmarkHostSeen(id string) error {
	m, err := loadSeenHosts()
	if err != nil {
		return err
	}
	delete(m, id)
	return saveSeenHosts(m)
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

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
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
	return atomicWriteFile(path, append(raw, 10), 0o600)
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

// ForgetSSHHost removes TOFU pin(s). Accepts:
//   - "host|port" (port-aware id)
//   - "host:port" / "[ipv6]:port" (OpenSSH-style)
//   - bare "host" — removes legacy host entry and every host|* pin for that host
//
// Missing entries are not an error (safe for re-provision).
func ForgetSSHHost(id string) error {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	f, err := loadRelayKH()
	if err != nil {
		return err
	}
	host, port, specific := parseForgetTarget(id)
	if specific {
		ep := endpointID(host, port)
		// Refuse when a legacy bare-host pin exists: lookupPin would keep trusting
		// this host on every port. Operator must forget the bare host (clears all).
		if leg := strings.TrimSpace(f[host]); leg != "" {
			return fmt.Errorf("refuse forget %s: legacy pin for %s still present (lookup would keep trusting this host) — run: netductor ssh-hosts forget %s", ep, host, host)
		}
		delete(f, ep)
		if err := saveRelayKH(f); err != nil {
			return err
		}
		if err := unmarkHostSeen(ep); err != nil {
			return err
		}
		return nil
	}
	// bare host: remove legacy + all port-aware keys for host
	delete(f, host)
	prefix := host + "|"
	var keys []string
	for k := range f {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		delete(f, k)
	}
	if err := saveRelayKH(f); err != nil {
		return err
	}
	var first error
	for _, k := range keys {
		if err := unmarkHostSeen(k); err != nil && first == nil {
			first = err
		}
	}
	if err := unmarkHostSeen(host); err != nil && first == nil {
		first = err
	}
	return first
}

// parseForgetTarget returns host, port, and whether a specific port was requested.
func parseForgetTarget(id string) (host, port string, specific bool) {
	id = strings.TrimSpace(id)
	if i := strings.Index(id, "|"); i >= 0 {
		return normalizeHost(id[:i]), strings.TrimSpace(id[i+1:]), true
	}
	if h, p, err := net.SplitHostPort(id); err == nil {
		return normalizeHost(h), p, true
	}
	return normalizeHost(id), "", false
}

// ClearSSHHosts removes all secondary provision TOFU keys.
func ClearSSHHosts() error {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	if err := saveRelayKH(map[string]string{}); err != nil {
		return err
	}
	return saveSeenHosts(map[string]bool{})
}

func provisionHostKey(host string, port int) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		pkhMu.Lock()
		defer pkhMu.Unlock()
		f, err := loadRelayKH()
		if err != nil {
			return err
		}
		got := base64.StdEncoding.EncodeToString(key.Marshal())
		portStr := "22"
		if port > 0 {
			portStr = fmt.Sprintf("%d", port)
		}
		host = normalizeHost(host)
		id := endpointID(host, portStr)
		if err := ensureSeenMigratedFromPins(f); err != nil {
			return err
		}
		if prevID, prev, ok := lookupPin(f, host, portStr); ok {
			if prev != got {
				return fmt.Errorf("SSH host key changed for %s — forget via: netductor ssh-hosts forget --kind relay %s", prevID, prevID)
			}
			if prevID != id {
				f[id] = got
				if err := saveRelayKH(f); err != nil {
					return err
				}
			}
			return markHostSeen(id)
		}
		seen, err := loadSeenHosts()
		if err != nil {
			return err
		}
		if pinLostForEndpoint(seen, f, host, portStr) {
			return fmt.Errorf("%w: %s — run: netductor ssh-hosts forget --kind relay %s then re-provision", ErrPinLost, id, id)
		}
		f[id] = got
		if err := saveRelayKH(f); err != nil {
			return err
		}
		return markHostSeen(id)
	}
}

// OpenSSHKnownHostsLine builds a StrictHostKeyChecking=yes line from TOFU store.
// Empty string if host not yet pinned (first contact still uses accept-new only when no pin).
func OpenSSHKnownHostsLine(host, port string) (string, error) {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	host = normalizeHost(host)
	if strings.TrimSpace(port) == "" {
		port = "22"
	}
	f, err := loadRelayKH()
	if err != nil {
		return "", err
	}
	if err := ensureSeenMigratedFromPins(f); err != nil {
		return "", err
	}
	id, b64, ok := lookupPin(f, host, port)
	if !ok || b64 == "" {
		seen, err := loadSeenHosts()
		if err != nil {
			return "", err
		}
		if pinLostForEndpoint(seen, f, host, port) {
			return "", fmt.Errorf("%w: %s — run: netductor ssh-hosts forget --kind relay %s then re-provision (explicit first contact)", ErrPinLost, id, id)
		}
		return "", fmt.Errorf("no TOFU pin for %s — run provision SSH first", id)
	}
	// Pin present: mark endpoint only (not bare host — preserves port independence).
	if err := markHostSeen(id); err != nil {
		return "", err
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

// MigrateSSHTrustSeen ensures all existing TOFU pins are recorded in ssh_hosts_seen.json.
// Safe and idempotent; call on upgrade or doctor.
func MigrateSSHTrustSeen() error {
	pkhMu.Lock()
	defer pkhMu.Unlock()
	f, err := loadRelayKH()
	if err != nil {
		return err
	}
	return ensureSeenMigratedFromPins(f)
}
