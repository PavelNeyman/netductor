package install

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"

	"github.com/PavelNeyman/netductor/internal/secondary"
	"strings"
)

// Default service-plane CIDRs (dual WG-over-WSS + legacy backbone).
var defaultAPIAllowCIDRs = []string{
	"10.87.10.0/30", // nd-svc-sp
	"10.87.11.0/30", // nd-svc-ps
	"10.87.0.0/30",  // legacy nd-backbone
	"127.0.0.1/32",
}

// ApplyAgentFirewall: deny plain :8788; restrict mTLS :8789.
//
// Default (locked): allow only service-plane CIDRs + /etc/netductor/api-allow.cidr
// and NETDUCTOR_API_ALLOW_CIDR (comma-separated).
// Temporary WAN open: only via `netductor api-public arm` (TG/CLI), not env.
// Operator steady-state: SSH tunnel or service CIDR / api-allow.cidr.
func ApplyAgentFirewall() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = run("ufw", "delete", "allow", "8788/tcp")
	_ = run("ufw", "deny", "8788/tcp")
	// wipe broad allows (v4/v6)
	_ = run("ufw", "delete", "allow", "8789/tcp")
	_ = run("ufw", "delete", "allow", "8789/tcp")

	cidrs := append([]string{}, defaultAPIAllowCIDRs...)
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

	seen := map[string]bool{}
	for _, c := range cidrs {
		if seen[c] {
			continue
		}
		seen[c] = true
		// ufw: allow from CIDR to port
		if err := run("ufw", "allow", "from", c, "to", "any", "port", "8789", "proto", "tcp", "comment", "netductor-api"); err != nil {
			fmt.Fprintf(os.Stderr, "ufw allow 8789 from %s: %v\n", c, err)
		}
	}
	fmt.Fprintln(os.Stderr, "ufw: :8788 denied, :8789 restricted to service CIDRs + api-allow.cidr (WAN open only via api-public arm)")
	return nil
}

// RestrictAgentMTLSToIP adds a single IP to the allow path then re-applies locked policy.
func RestrictAgentMTLSToIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip != "" && !strings.Contains(ip, "/") {
		ip += "/32"
	}
	if ip != "" {
		_ = os.MkdirAll("/etc/netductor", 0o755)
		// append if not present
		b, _ := os.ReadFile("/etc/netductor/api-allow.cidr")
		if !strings.Contains(string(b), ip) {
			f, err := os.OpenFile("/etc/netductor/api-allow.cidr", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err == nil {
				_, _ = fmt.Fprintln(f, ip)
				_ = f.Close()
			}
		}
	}
	return ApplyAgentFirewall()
}

// AllowAgentMTLSFromIP adds operator/path CIDR and re-applies.
func AllowAgentMTLSFromIP(ip string) error {
	return RestrictAgentMTLSToIP(ip)
}


// applyAgentFirewallQuiet is ApplyAgentFirewall without ufw chatter on stderr.
func applyAgentFirewallQuiet() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = ufwQuiet("delete", "allow", "8788/tcp")
	_ = ufwQuiet("deny", "8788/tcp")
	// do not wipe all 8789 rules — would remove service CIDRs; only ensure service CIDRs exist
	cidrs := append([]string{}, defaultAPIAllowCIDRs...)
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
	seen := map[string]bool{}
	for _, c := range cidrs {
		if seen[c] {
			continue
		}
		seen[c] = true
		_ = ufwQuiet("allow", "from", c, "to", "any", "port", "8789", "proto", "tcp", "comment", "netductor-api")
	}
	return nil
}

// SyncAgentAllowFromSecondaryRegistry writes secondary public IPs into api-allow.cidr.
func SyncAgentAllowFromSecondaryRegistry() error {
	for _, ip := range secondary.ListPublicIPs() {
		_ = RestrictAgentMTLSToIP(ip) // append + apply each time is ok
	}
	return ApplyAgentFirewall()
}
