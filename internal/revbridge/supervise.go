package revbridge

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// In-process supervisors for primary (client) and secondary (server).

var (
	supMu     sync.Mutex
	server    *Server
	client    *Client
	clientHost string
)

// StartServerSecondary starts public+local listeners for given kinds (no wg).
func StartServerSecondary(kinds []Kind) error {
	supMu.Lock()
	defer supMu.Unlock()
	if server != nil {
		server.Stop()
		server = nil
	}
	filtered := nonWG(kinds)
	if len(filtered) == 0 {
		return nil
	}
	s := NewServer(filtered)
	if err := s.Start(); err != nil {
		return err
	}
	server = s
	fmt.Fprintf(os.Stderr, "revbridge: secondary server kinds=%v\n", filtered)
	return nil
}

// StartClientPrimary dials secondaryHost for given kinds.
func StartClientPrimary(secondaryHost string, kinds []Kind) error {
	secondaryHost = strings.TrimSpace(secondaryHost)
	if secondaryHost == "" {
		return fmt.Errorf("revbridge: empty secondary host")
	}
	supMu.Lock()
	defer supMu.Unlock()
	if client != nil {
		client.Stop()
		client = nil
	}
	filtered := nonWG(kinds)
	if len(filtered) == 0 {
		return nil
	}
	cl := NewClient(secondaryHost, filtered)
	cl.Start()
	client = cl
	clientHost = secondaryHost
	fmt.Fprintf(os.Stderr, "revbridge: primary client → %s kinds=%v\n", secondaryHost, filtered)
	return nil
}

func StopAll() {
	supMu.Lock()
	defer supMu.Unlock()
	if server != nil {
		server.Stop()
		server = nil
	}
	if client != nil {
		client.Stop()
		client = nil
	}
}

func nonWG(kinds []Kind) []Kind {
	var out []Kind
	for _, k := range kinds {
		if k != KindWG {
			out = append(out, k)
		}
	}
	return out
}

func Status() map[string]any {
	supMu.Lock()
	defer supMu.Unlock()
	return map[string]any{
		"server_running": server != nil,
		"client_running": client != nil,
		"client_host":    clientHost,
	}
}
