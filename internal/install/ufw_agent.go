package install

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

var apiAllowFileMu sync.Mutex

// defaultAPIAllowCIDRs from ndconfig (SP/PS service paths only).
func defaultAPIAllowCIDRs() []string {
	return []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1/32"}
}

// ApplyAgentFirewall: deny plain :8788; restrict mTLS :8789.
func ApplyAgentFirewall() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	var cidrs []string
	if os.Getenv("NETDUCTOR_API_ALLOW_PUBLIC") != "1" {
		var err error
		cidrs, err = collectAPIAllowCIDRs()
		if err != nil {
			return fmt.Errorf("load API allow CIDRs: %w", err)
		}
		if err := validateAPIAllowCIDRs(cidrs); err != nil {
			return err
		}
	}

	// Configuration is fully read and validated before the first firewall mutation.
	_ = run("ufw", "delete", "allow", "8788/tcp")
	if err := run("ufw", "deny", "8788/tcp"); err != nil {
		return fmt.Errorf("ufw deny 8788/tcp: %w", err)
	}
	_ = run("ufw", "delete", "allow", ndconfig.AgentMTLSPort()+"/tcp")
	_ = run("ufw", "delete", "allow", ndconfig.AgentMTLSPort()+"/tcp")

	if os.Getenv("NETDUCTOR_API_ALLOW_PUBLIC") == "1" {
		if err := run("ufw", "allow", ndconfig.AgentMTLSPort()+"/tcp", "comment", "netductor-api-public"); err != nil {
			return fmt.Errorf("ufw allow 8789 public: %w", err)
		}
		fmt.Fprintln(os.Stderr, "ufw: :8788 denied, :8789 OPEN (NETDUCTOR_API_ALLOW_PUBLIC=1)")
		return nil
	}

	seen := map[string]bool{}
	for _, c := range cidrs {
		if seen[c] { continue }
		seen[c] = true
		if err := run("ufw", "allow", "from", c, "to", "any", "port", ndconfig.AgentMTLSPort(), "proto", "tcp", "comment", "netductor-api"); err != nil {
			return fmt.Errorf("ufw allow mTLS from %s: %w", c, err)
		}
	}
	fmt.Fprintln(os.Stderr, "ufw: :8788 denied, :8789 restricted to service CIDRs + api-allow.cidr")
	return nil
}

func collectAPIAllowCIDRs() ([]string, error) {
	cidrs := append([]string{}, defaultAPIAllowCIDRs()...)
	if extra := strings.TrimSpace(os.Getenv("NETDUCTOR_API_ALLOW_CIDR")); extra != "" {
		for _, p := range strings.Split(extra, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cidrs = append(cidrs, p)
			}
		}
	}
	f, err := os.Open("/etc/netductor/api-allow.cidr")
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("open /etc/netductor/api-allow.cidr: %w", err)
		}
		return cidrs, nil
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cidrs = append(cidrs, line)
	}
	scanErr := sc.Err()
	closeErr := f.Close()
	if scanErr != nil {
		return nil, fmt.Errorf("scan /etc/netductor/api-allow.cidr: %w", scanErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close /etc/netductor/api-allow.cidr: %w", closeErr)
	}
	return cidrs, nil
}

func validateAPIAllowCIDRs(cidrs []string) error {
	seen := make(map[string]bool, len(cidrs))
	for _, c := range cidrs {
		if seen[c] {
			continue
		}
		if _, _, err := net.ParseCIDR(c); err != nil {
			return fmt.Errorf("invalid API allow CIDR %q: %w", c, err)
		}
		seen[c] = true
	}
	return nil
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
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return fmt.Errorf("invalid agent public IP %q", ip)
		}
		if parsed.To4() != nil {
			ip += "/32"
		} else {
			ip += "/128"
		}
	}
	_, network, err := net.ParseCIDR(ip)
	if err != nil {
		return fmt.Errorf("invalid agent allow CIDR %q: %w", ip, err)
	}
	ip = network.String()

	apiAllowFileMu.Lock()
	defer apiAllowFileMu.Unlock()
	if cidrAlreadyListed(ip) {
		return nil
	}
	if err := os.MkdirAll("/etc/netductor", 0o755); err != nil {
		return fmt.Errorf("create /etc/netductor: %w", err)
	}
	f, err := os.OpenFile("/etc/netductor/api-allow.cidr", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open api-allow.cidr: %w", err)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fmt.Errorf("chmod api-allow.cidr: %w", err)
	}
	if _, err := fmt.Fprintln(f, ip); err != nil {
		_ = f.Close()
		return fmt.Errorf("append api-allow.cidr: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync api-allow.cidr: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close api-allow.cidr: %w", err)
	}
	// Apply the new rule only after the allowlist entry is durable.
	if _, err := exec.LookPath("ufw"); err == nil {
		if err := run("ufw", "allow", "from", ip, "to", "any", "port", ndconfig.AgentMTLSPort(), "proto", "tcp", "comment", "netductor-api"); err != nil {
			return fmt.Errorf("ufw allow mTLS from %s: %w", ip, err)
		}
	}
	return nil
}

// AllowAgentMTLSFromIP adds operator/path CIDR without hammering ufw.

func applyAgentFirewallQuiet() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	cidrs, err := collectAPIAllowCIDRs()
	if err != nil {
		return fmt.Errorf("load API allow CIDRs: %w", err)
	}
	if err := validateAPIAllowCIDRs(cidrs); err != nil {
		return err
	}
	_ = ufwQuiet("delete", "allow", "8788/tcp") // absent rule is expected
	if err := ufwQuiet("deny", "8788/tcp"); err != nil {
		return fmt.Errorf("ufw deny 8788/tcp: %w", err)
	}
	seen := map[string]bool{}
	for _, c := range cidrs {
		if seen[c] {
			continue
		}
		seen[c] = true
		if err := ufwQuiet("allow", "from", c, "to", "any", "port", ndconfig.AgentMTLSPort(), "proto", "tcp", "comment", "netductor-api"); err != nil {
			return fmt.Errorf("ufw allow mTLS from %s: %w", c, err)
		}
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
