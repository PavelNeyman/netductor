package operator

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TunnelOpts struct {
	Host     string
	User     string
	Key      string
	KeyPass  string // unused for BatchMode ssh; document passphrase keys need ssh-agent
	LocalPort string
	SSHPort  string
}

type tunnelState struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	opts   TunnelOpts
	since  time.Time
	lastErr string
}

var globalTunnel tunnelState

func expandKeyPath(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".ssh", "netductor_primary")
	}
	if strings.HasPrefix(key, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, key[2:])
	}
	return key
}

func (o *TunnelOpts) normalize() error {
	o.Host = strings.TrimSpace(o.Host)
	if o.Host == "" {
		return fmt.Errorf("host required")
	}
	if o.User == "" {
		o.User = "root"
	}
	o.Key = expandKeyPath(o.Key)
	if o.LocalPort == "" {
		o.LocalPort = "8787"
	}
	if o.SSHPort == "" {
		o.SSHPort = strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
	}
	if o.SSHPort == "" {
		o.SSHPort = "52222"
	}
	return nil
}

// TunnelStart starts SSH -L if not already running to same target.
func TunnelStart(o TunnelOpts) error {
	if err := o.normalize(); err != nil {
		return err
	}
	globalTunnel.mu.Lock()
	defer globalTunnel.mu.Unlock()
	if globalTunnel.cmd != nil && globalTunnel.cmd.Process != nil {
		// already up — same host/port ok
		if globalTunnel.opts.Host == o.Host && globalTunnel.opts.LocalPort == o.LocalPort {
			return nil
		}
		_ = globalTunnel.cmd.Process.Kill()
		globalTunnel.cmd = nil
	}
	fwd := fmt.Sprintf("%s:127.0.0.1:8787", o.LocalPort)
	argv := []string{
		"-N", "-L", fwd,
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", o.SSHPort,
		"-i", o.Key,
		"--", o.User + "@" + o.Host,
	}
	cmd := exec.Command("ssh", argv...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		globalTunnel.lastErr = err.Error()
		return err
	}
	globalTunnel.cmd = cmd
	globalTunnel.opts = o
	globalTunnel.since = time.Now()
	globalTunnel.lastErr = ""
	// wait briefly for port
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+o.LocalPort, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			go func() {
				_ = cmd.Wait()
				globalTunnel.mu.Lock()
				if globalTunnel.cmd == cmd {
					globalTunnel.cmd = nil
				}
				globalTunnel.mu.Unlock()
			}()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	// process may still be connecting — leave running
	go func() {
		_ = cmd.Wait()
		globalTunnel.mu.Lock()
		if globalTunnel.cmd == cmd {
			globalTunnel.cmd = nil
		}
		globalTunnel.mu.Unlock()
	}()
	return nil
}

func TunnelStop() {
	globalTunnel.mu.Lock()
	defer globalTunnel.mu.Unlock()
	if globalTunnel.cmd != nil && globalTunnel.cmd.Process != nil {
		_ = globalTunnel.cmd.Process.Kill()
	}
	globalTunnel.cmd = nil
}

func TunnelStatus() map[string]any {
	globalTunnel.mu.Lock()
	defer globalTunnel.mu.Unlock()
	up := globalTunnel.cmd != nil && globalTunnel.cmd.Process != nil
	portOpen := false
	port := globalTunnel.opts.LocalPort
	if port == "" {
		port = "8787"
	}
	if c, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 150*time.Millisecond); err == nil {
		_ = c.Close()
		portOpen = true
	}
	m := map[string]any{
		"process_up": up,
		"port_open":  portOpen,
		"local_port": port,
		"host":       globalTunnel.opts.Host,
		"since":      "",
		"last_error": globalTunnel.lastErr,
	}
	if !globalTunnel.since.IsZero() {
		m["since"] = globalTunnel.since.UTC().Format(time.RFC3339)
	}
	return m
}
