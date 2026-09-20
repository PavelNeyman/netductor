package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Agent plane policy (default):
//   - :8788 denied always
//   - :8789 open to the world, protected by mTLS client certs
//   - secondary public IPs are recorded in agent_allowlist (inventory / optional strict mode)
//
// Edge routers sit behind ISP NAT with changing WAN IPs — IP allowlist must NOT gate them.
// Set NETDUCTOR_AGENT_ALLOWLIST_STRICT=1 to restrict :8789 to listed IPs only (secondary-only fleets).

func AgentAllowlistPath() string {
	return filepath.Join(paths.EtcDir(), "secrets", "agent_allowlist")
}

func AgentAllowlistStrict() bool {
	return os.Getenv("NETDUCTOR_AGENT_ALLOWLIST_STRICT") == "1"
}

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
	b.WriteString("# Secondary (stable public) IPs for agent plane inventory.\n")
	b.WriteString("# Edge/NAT devices are NOT listed — they use mTLS only.\n")
	b.WriteString("# NETDUCTOR_AGENT_ALLOWLIST_STRICT=1 → ufw allows only these IPs on :8789.\n")
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

// RecordAgentAllowlistIP stores a stable agent IP (typically secondary). Does not close the port for others unless STRICT.
func RecordAgentAllowlistIP(ip string) error {
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

// AllowAgentMTLSFromIP records secondary IP and applies firewall policy.
func AllowAgentMTLSFromIP(ip string) error {
	return RecordAgentAllowlistIP(ip)
}

// RestrictAgentMTLSToIP keeps API compatibility with secondary provision.
func RestrictAgentMTLSToIP(ip string) error {
	return RecordAgentAllowlistIP(ip)
}

// ApplyAgentAllowlistFirewall:
// default — deny 8788, allow 8789 from anywhere (mTLS is the gate);
// STRICT  — deny 8788, allow 8789 only from allowlisted IPs.
func ApplyAgentAllowlistFirewall() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = run("ufw", "delete", "allow", "8788/tcp")
	_ = run("ufw", "deny", "8788/tcp")
	_ = run("ufw", "delete", "allow", "8789/tcp")

	if AgentAllowlistStrict() {
		ips := LoadAgentAllowlist()
		if len(ips) == 0 {
			fmt.Fprintln(os.Stderr, "ufw STRICT: allowlist empty — :8789 not opened")
			return nil
		}
		for _, ip := range ips {
			if err := run("ufw", "allow", "from", ip, "to", "any", "port", "8789", "proto", "tcp"); err != nil {
				fmt.Fprintf(os.Stderr, "ufw: allow 8789 from %s: %v\n", ip, err)
			} else {
				fmt.Fprintf(os.Stderr, "ufw STRICT: :8789 from %s\n", ip)
			}
		}
		return nil
	}

	// Default: edge behind NAT needs :8789 reachable; auth = mTLS client cert.
	if err := run("ufw", "allow", "8789/tcp"); err != nil {
		return fmt.Errorf("ufw allow 8789: %w", err)
	}
	fmt.Fprintln(os.Stderr, "ufw: :8789 open (mTLS required); secondary IPs recorded in agent_allowlist")
	return nil
}
