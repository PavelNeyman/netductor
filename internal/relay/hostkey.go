package relay

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

var pkhMu sync.Mutex

func provisionHostKey(host string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		pkhMu.Lock()
		defer pkhMu.Unlock()
		path := filepath.Join(paths.StateDir(), "relay", "ssh_known_hosts.json")
		f := map[string]string{}
		if b, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(b, &f)
		}
		got := base64.StdEncoding.EncodeToString(key.Marshal())
		id := host
		if prev, ok := f[id]; ok {
			if prev != got {
				return fmt.Errorf("SSH host key changed for %s", id)
			}
			return nil
		}
		f[id] = got
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		raw, _ := json.MarshalIndent(f, "", "  ")
		return os.WriteFile(path, append(raw, 10), 0o600)
	}
}
