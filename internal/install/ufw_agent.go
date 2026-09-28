package install

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

// defaultAPIAllowCIDRs from ndconfig (overridable). Legacy backbone only if NETDUCTOR_SVC_LEGACY_CIDR set.
func defaultAPIAllowCIDRs() []string {
	out := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1/32"}
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_SVC_LEGACY_CIDR")); v != "" {
		out = append(out, v)
	}
	return out
}

// ApplyAgentFirewall: deny plain :8788; restrict mTLS :8789.
func ApplyAgentFirewall() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = run("ufw", "delete", "allow", "8788/tcp")
	_ = run("ufw", "deny", "8788/tcp")
	_ = run("ufw", "delete", "allow", ndconfig.AgentMTLSPort()+"/tcp")
	_ = run("ufw", "delete", "allow", ndconfig.AgentMTLSPort()+"/tcp")

	if os.Getenv("NETDUCTOR_API_ALLOW_PUBLIC") == "1" {
		if err := run("ufw", "allow", ndconfig.AgentMTLSPort()+"/tcp", "comment", "netductor-api-public"); err != nil {
			return fmt.Errorf("ufw allow 8789 public: %w", err)
		}
		fmt.Fprintln(os.Stderr, "ufw: :8788 denied, :8789 OPEN (NETDUCTOR_API_ALLOW_PUBLIC=1)")
		return nil
	}

	cidrs := collectAPIAllowCIDRs()
	seen := map[string]bool{}
	for _, c := range cidrs {
		if seen[c] {
			continue
		}
		seen[c] = true
		if err := run("ufw", "allow", "from", c, "to", "any", "port", ndconfig.AgentMTLSPort(), "proto", "tcp", "comment", "netductor-api"); err != nil {
			fmt.Fprintf(os.Stderr, "ufw allow 8789 from %s: %v\n", c, err)
		}
	}
	fmt.Fprintln(os.Stderr, "ufw: :8788 denied, :8789 restricted to service CIDRs + api-allow.cidr")
	return nil
}

func collectAPIAllowCIDRs() []string {
	cidrs := append([]string{}, defaultAPIAllowCIDRs()...)
	if extra := strings.TrimSpace(os.Getenv("NETDUCTOR_API_ALLOW_CIDR")); extra != "" {
		for _, p := range strings.Split(extra, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cidrs = append(cidrs, p)
			}
		}
	}
	if f, err := os.Open("/etc/netductor/api-allow.cidr"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			cidrs = append(cidrs, line)
		}
		_ = f.Close()
	}
	return cidrs
}

// cidrAlreadyListed reports whether ip/CIDR is already in api-allow.cidr (exact line).
func cidrAlreadyListed(cidr string) bool {
	b, err := os.ReadFile("/etc/netductor/api-allow.cidr")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == cidr {
			return true
		}
	}
	return false
}

// RestrictAgentMTLSToIP appends IP to api-allow.cidr once. Does NOT re-apply full ufw
// when the IP is already listed (heartbeat used to spam ufw every 30s).
func RestrictAgentMTLSToIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	if !strings.Contains(ip, "/") {
		ip += "/32"
	}
	if cidrAlreadyListed(ip) {
		return nil
	}
	_ = os.MkdirAll("/etc/netductor", 0o755)
	f, err := os.OpenFile("/etc/netductor/api-allow.cidr", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(f, ip)
	_ = f.Close()
	// one-shot ufw allow for this CIDR only (no full wipe/reapply)
	if _, err := exec.LookPath("ufw"); err == nil {
		_ = run("ufw", "allow", "from", ip, "to", "any", "port", ndconfig.AgentMTLSPort(), "proto", "tcp", "comment", "netductor-api")
	}
	return nil
}

// AllowAgentMTLSFromIP adds operator/path CIDR without hammering ufw.
func AllowAgentMTLSFromIP(ip string) error {
	return RestrictAgentMTLSToIP(ip)
}

func applyAgentFirewallQuiet() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = ufwQuiet("delete", "allow", "8788/tcp")
	_ = ufwQuiet("deny", "8788/tcp")
	seen := map[string]bool{}
	for _, c := range collectAPIAllowCIDRs() {
		if seen[c] {
			continue
		}
		seen[c] = true
		_ = ufwQuiet("allow", "from", c, "to", "any", "port", ndconfig.AgentMTLSPort(), "proto", "tcp", "comment", "netductor-api")
	}
	return nil
}

// SyncAgentAllowFromSecondaryRegistry writes secondary public IPs into api-allow.cidr.
func SyncAgentAllowFromSecondaryRegistry() error {
	for _, ip := range secondary.ListPublicIPs() {
		_ = RestrictAgentMTLSToIP(ip)
	}
	return nil
}

