package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// AgentAllowlistPath stores IPs allowed to reach mTLS agent plane :8789.
func AgentAllowlistPath() string {
	return filepath.Join(paths.EtcDir(), "secrets", "agent_allowlist")
}

// LoadAgentAllowlist returns unique non-empty IPs.
func LoadAgentAllowlist() []string {
	b, err := os.ReadFile(AgentAllowlistPath())
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		ip := strings.TrimSpace(line)
		if ip == "" || strings.HasPrefix(ip, "#") {
			continue
		}
		if seen[ip] {
			continue
		}
		seen[ip] = true
		out = append(out, ip)
	}
	return out
}

func saveAgentAllowlist(ips []string) error {
	dir := filepath.Dir(AgentAllowlistPath())
	_ = os.MkdirAll(dir, 0o700)
	var b strings.Builder
	b.WriteString("# netductor agent plane :8789 allowlist (one IP per line)\n")
	seen := map[string]bool{}
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" || seen[ip] {
			continue
		}
		seen[ip] = true
		b.WriteString(ip)
		b.WriteByte('\n')
	}
	return os.WriteFile(AgentAllowlistPath(), []byte(b.String()), 0o600)
}

// AllowAgentMTLSFromIP adds IP to allowlist and reapplies ufw (default policy).
func AllowAgentMTLSFromIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return fmt.Errorf("ip required")
	}
	ips := LoadAgentAllowlist()
	for _, x := range ips {
		if x == ip {
			return ApplyAgentAllowlistFirewall()
		}
	}
	ips = append(ips, ip)
	if err := saveAgentAllowlist(ips); err != nil {
		return err
	}
	return ApplyAgentAllowlistFirewall()
}

// RestrictAgentMTLSToIP is an alias for AllowAgentMTLSFromIP (multi-IP allowlist is default).
func RestrictAgentMTLSToIP(ip string) error {
	return AllowAgentMTLSFromIP(ip)
}

// ApplyAgentAllowlistFirewall makes ufw default for agent plane:
// - deny 8788 always
// - delete broad allow 8789
// - allow 8789 only from each allowlisted IP
// If allowlist is empty, 8789 stays closed (no world open).
func ApplyAgentAllowlistFirewall() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = run("ufw", "delete", "allow", "8788/tcp")
	_ = run("ufw", "deny", "8788/tcp")
	_ = run("ufw", "delete", "allow", "8789/tcp")
	// Remove previous from-IP rules is hard; re-allow listed IPs (ufw dedupes).
	ips := LoadAgentAllowlist()
	if len(ips) == 0 {
		fmt.Fprintln(os.Stderr, "ufw: agent :8789 closed (empty allowlist — add IP on secondary/edge provision)")
		return nil
	}
	for _, ip := range ips {
		if err := run("ufw", "allow", "from", ip, "to", "any", "port", "8789", "proto", "tcp"); err != nil {
			fmt.Fprintf(os.Stderr, "ufw: allow 8789 from %s: %v\n", ip, err)
		} else {
			fmt.Fprintf(os.Stderr, "ufw: agent mTLS :8789 allowed from %s\n", ip)
		}
	}
	return nil
}
