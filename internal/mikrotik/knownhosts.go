package mikrotik

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var khMu sync.Mutex

func knownHostsPath() string {
	return filepath.Join(paths.StateDir(), "mikrotik", "known_hosts.json")
}

type khFile struct {
	Hosts map[string]string `json:"hosts"` // host:port -> base64(ssh.MarshalAuthorizedKey style raw key)
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

func hostKeyKey(host string, port int, key ssh.PublicKey) string {
	return fmt.Sprintf("%s:%d", host, port)
}

// hostKeyCallback TOFU: first connect stores key; later verifies.
// NETDUCTOR_MT_STRICT=1 rejects unknown hosts instead of storing.
func hostKeyCallback(host string, port int) ssh.HostKeyCallback {
	strict := os.Getenv("NETDUCTOR_MT_STRICT") == "1"
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		khMu.Lock()
		defer khMu.Unlock()
		f := loadKH()
		id := hostKeyKey(host, port, key)
		got := base64.StdEncoding.EncodeToString(key.Marshal())
		if prev, ok := f.Hosts[id]; ok {
			if prev != got {
				return fmt.Errorf("host key mismatch for %s (TOFU) — remove entry in %s if MT was reinstalled", id, knownHostsPath())
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
