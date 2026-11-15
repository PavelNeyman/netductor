package revbridge

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	supMu      sync.Mutex
	server     *Server
	client     *Client
	clientHost string
	serverKey  string // kinds+allowFrom signature
	clientKey  string
)

func kindsKey(kinds []Kind, extra string) string {
	var b strings.Builder
	for _, k := range kinds {
		if k == KindWG {
			continue
		}
		b.WriteString(string(k))
		b.WriteByte(',')
	}
	b.WriteString("|")
	b.WriteString(extra)
	return b.String()
}

// StartServerSecondary starts public+local listeners for given kinds (no wg).
// No-op if already running with the same kinds and allowFrom (avoids session flaps).
func StartServerSecondary(kinds []Kind, allowFrom string) error {
	supMu.Lock()
	defer supMu.Unlock()
	filtered := nonWG(kinds)
	key := kindsKey(filtered, strings.TrimSpace(allowFrom))
	if server != nil && serverKey == key {
		return nil
	}
	if server != nil {
		server.Stop()
		server = nil
		serverKey = ""
	}
	if len(filtered) == 0 {
		return nil
	}
	s := NewServer(filtered)
	s.SetAllowFrom(allowFrom)
	if err := s.Start(); err != nil {
		return err
	}
	server = s
	serverKey = key
	fmt.Fprintf(os.Stderr, "revbridge: secondary server kinds=%v allowFrom=%q\n", filtered, allowFrom)
	return nil
}

// StartClientPrimary dials secondaryHost for given kinds.
// No-op if already dialing the same host+kinds.
func StartClientPrimary(secondaryHost string, kinds []Kind) error {
	secondaryHost = strings.TrimSpace(secondaryHost)
	supMu.Lock()
	defer supMu.Unlock()
	filtered := nonWG(kinds)
	key := kindsKey(filtered, secondaryHost)
	if client != nil && clientKey == key {
		return nil
	}
	if client != nil {
		client.Stop()
		client = nil
		clientHost = ""
		clientKey = ""
	}
	if len(filtered) == 0 {
		return nil
	}
	if secondaryHost == "" {
		return fmt.Errorf("revbridge: empty secondary host")
	}
	cl := NewClient(secondaryHost, filtered)
	cl.Start()
	client = cl
	clientHost = secondaryHost
	clientKey = key
	fmt.Fprintf(os.Stderr, "revbridge: primary client → %s kinds=%v\n", secondaryHost, filtered)
	return nil
}

func StopAll() {
	supMu.Lock()
	defer supMu.Unlock()
	if server != nil {
		server.Stop()
		server = nil
		serverKey = ""
	}
	if client != nil {
		client.Stop()
		client = nil
		clientHost = ""
		clientKey = ""
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
		"note":           "in-process only; CLI status is always empty — check journalctl -u netductor-api",
	}
}
