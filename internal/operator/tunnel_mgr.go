package operator

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TunnelOpts struct {
	Host      string
	VPNHost   string // preferred SSH target when PreferVPN and reachable
	PreferVPN bool
	User      string
	Key       string
	KeyPass   string
	LocalPort string
	SSHPort   string
}

type tunnelState struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	opts    TunnelOpts
	since   time.Time
	lastErr string
	via     string // "vpn" | "public" | ""
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
	o.VPNHost = strings.TrimSpace(o.VPNHost)
	if o.Host == "" && o.VPNHost == "" {
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

// sshHostReachable quick TCP check to host:sshPort.
func sshHostReachable(host, port string) bool {
	if host == "" {
		return false
	}
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 800*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// ResolveSSHHost prefers VPN host when PreferVPN and TCP to SSH port works.
func ResolveSSHHost(o TunnelOpts) (host, via string) {
	_ = o.normalize()
	if o.PreferVPN && o.VPNHost != "" && sshHostReachable(o.VPNHost, o.SSHPort) {
		return o.VPNHost, "vpn"
	}
	if o.Host != "" {
		return o.Host, "public"
	}
	if o.VPNHost != "" {
		return o.VPNHost, "vpn-fallback"
	}
	return "", ""
}

// APIReachable probes node health without tunnel requirement (existing tunnel or VPN-routed base).
func APIReachable(apiBase string) bool {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if apiBase == "" {
		apiBase = "http://127.0.0.1:8787"
	}
	client := &http.Client{Timeout: 900 * time.Millisecond}
	resp, err := client.Get(apiBase + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// EnsureAPIPath: if API already up on local base → no tunnel; else SSH tunnel (VPN host preferred).
func EnsureAPIPath(o TunnelOpts, apiBase string) (map[string]any, error) {
	if APIReachable(apiBase) {
		return map[string]any{
			"ok": true, "path": "direct", "tunnel": false,
			"api_base": apiBase, "status": TunnelStatus(),
			"note": "API reachable without new tunnel (VPN or existing forward)",
		}, nil
	}
	if err := TunnelStart(o); err != nil {
		return map[string]any{"ok": false, "path": "ssh", "error": err.Error(), "status": TunnelStatus()}, err
	}
	return map[string]any{
		"ok": true, "path": "ssh", "tunnel": true,
		"via": globalTunnel.via, "status": TunnelStatus(),
	}, nil
}

func TunnelStart(o TunnelOpts) error {
	if err := o.normalize(); err != nil {
		return err
	}
	host, via := ResolveSSHHost(o)
	if host == "" {
		return fmt.Errorf("no reachable SSH host")
	}
	o.Host = host

	globalTunnel.mu.Lock()
	defer globalTunnel.mu.Unlock()
	if globalTunnel.cmd != nil && globalTunnel.cmd.Process != nil {
		if globalTunnel.opts.Host == o.Host && globalTunnel.opts.LocalPort == o.LocalPort {
			globalTunnel.via = via
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
	if err := cmd.Start(); err != nil {
		globalTunnel.lastErr = err.Error()
		return err
	}
	globalTunnel.cmd = cmd
	globalTunnel.opts = o
	globalTunnel.via = via
	globalTunnel.since = time.Now()
	globalTunnel.lastErr = ""
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+o.LocalPort, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			go waitTunnel(cmd)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	go waitTunnel(cmd)
	return nil
}

func waitTunnel(cmd *exec.Cmd) {
	_ = cmd.Wait()
	globalTunnel.mu.Lock()
	if globalTunnel.cmd == cmd {
		globalTunnel.cmd = nil
	}
	globalTunnel.mu.Unlock()
}

func TunnelStop() {
	globalTunnel.mu.Lock()
	defer globalTunnel.mu.Unlock()
	if globalTunnel.cmd != nil && globalTunnel.cmd.Process != nil {
		_ = globalTunnel.cmd.Process.Kill()
	}
	globalTunnel.cmd = nil
	globalTunnel.via = ""
}

func TunnelStatus() map[string]any {
	globalTunnel.mu.Lock()
	defer globalTunnel.mu.Unlock()
	up := globalTunnel.cmd != nil && globalTunnel.cmd.Process != nil
	port := globalTunnel.opts.LocalPort
	if port == "" {
		port = "8787"
	}
	portOpen := false
	if c, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 150*time.Millisecond); err == nil {
		_ = c.Close()
		portOpen = true
	}
	m := map[string]any{
		"process_up": up,
		"port_open":  portOpen,
		"local_port": port,
		"host":       globalTunnel.opts.Host,
		"via":        globalTunnel.via,
		"since":      "",
		"last_error": globalTunnel.lastErr,
	}
	if !globalTunnel.since.IsZero() {
		m["since"] = globalTunnel.since.UTC().Format(time.RFC3339)
	}
	return m
}
