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
	"golang.org/x/sys/unix"
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
	if err := deleteUFWUnrestrictedPort("8788"); err != nil {
		return fmt.Errorf("remove unrestricted 8788 allow rules: %w", err)
	}
	if err := run("ufw", "deny", "8788/tcp"); err != nil {
		return fmt.Errorf("ufw deny 8788/tcp: %w", err)
	}
	if err := deleteUFWUnrestrictedPort(ndconfig.AgentMTLSPort()); err != nil {
		return fmt.Errorf("remove unrestricted mTLS allow rules: %w", err)
	}

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
	f, err := openAPIAllowFile(unix.O_RDONLY, false)
	if err != nil {
		if os.IsNotExist(err) {
			return cidrs, nil
		}
		return nil, fmt.Errorf("open trusted api-allow.cidr: %w", err)
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

// openAPIAllowFile opens the root-owned allowlist without following symlinks.
// The containing directory and file must be root-owned and not group/world writable.
func openAPIAllowFile(flags int, create bool) (*os.File, error) {
	dirFD, err := unix.Open("/etc/netductor", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	var dirStat unix.Stat_t
	if err := unix.Fstat(dirFD, &dirStat); err != nil {
		_ = unix.Close(dirFD)
		return nil, fmt.Errorf("stat /etc/netductor: %w", err)
	}
	if dirStat.Mode&unix.S_IFMT != unix.S_IFDIR || dirStat.Uid != 0 || dirStat.Mode&0o022 != 0 {
		_ = unix.Close(dirFD)
		return nil, fmt.Errorf("/etc/netductor is not a trusted root-owned directory")
	}

	openFlags := flags | unix.O_CLOEXEC | unix.O_NOFOLLOW
	if create {
		openFlags |= unix.O_CREAT
	}
	fd, err := unix.Openat(dirFD, "api-allow.cidr", openFlags, 0o600)
	_ = unix.Close(dirFD)
	if err != nil {
		return nil, err
	}
	var fileStat unix.Stat_t
	if err := unix.Fstat(fd, &fileStat); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("stat api-allow.cidr: %w", err)
	}
	if fileStat.Mode&unix.S_IFMT != unix.S_IFREG || fileStat.Uid != 0 || fileStat.Mode&0o022 != 0 {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("api-allow.cidr is not a trusted root-owned regular file")
	}
	if flags&unix.O_WRONLY != 0 || flags&unix.O_RDWR != 0 {
		if err := unix.Fchmod(fd, 0o600); err != nil {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("chmod api-allow.cidr: %w", err)
		}
	}
	return os.NewFile(uintptr(fd), "/etc/netductor/api-allow.cidr"), nil
}

// cidrAlreadyListed reports whether ip/CIDR is already in api-allow.cidr.
func cidrAlreadyListed(cidr string) (bool, error) {
	f, err := openAPIAllowFile(unix.O_RDONLY, false)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == cidr {
			_ = f.Close()
			return true, nil
		}
	}
	scanErr := sc.Err()
	closeErr := f.Close()
	if scanErr != nil {
		return false, fmt.Errorf("scan api-allow.cidr: %w", scanErr)
	}
	if closeErr != nil {
		return false, fmt.Errorf("close api-allow.cidr: %w", closeErr)
	}
	return false, nil
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
	listed, err := cidrAlreadyListed(ip)
	if err != nil {
		return fmt.Errorf("check api-allow.cidr: %w", err)
	}
	if err := os.MkdirAll("/etc/netductor", 0o755); err != nil {
		return fmt.Errorf("create /etc/netductor: %w", err)
	}
	if !listed {
		f, err := openAPIAllowFile(unix.O_WRONLY|unix.O_APPEND, true)
		if err != nil {
			return fmt.Errorf("open trusted api-allow.cidr for append: %w", err)
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
	}
	// Reassert this one rule even for an existing entry: a prior UFW failure
	// must be retryable without re-appending the CIDR or rebuilding all rules.
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
	if err := deleteUFWUnrestrictedPort("8788"); err != nil {
		return fmt.Errorf("remove unrestricted 8788 allow rules: %w", err)
	}
	if err := ufwQuiet("deny", "8788/tcp"); err != nil {
		return fmt.Errorf("ufw deny 8788/tcp: %w", err)
	}
	if err := deleteUFWUnrestrictedPort(ndconfig.AgentMTLSPort()); err != nil {
		return fmt.Errorf("remove unrestricted mTLS allow rules: %w", err)
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
	ips, err := secondary.ListPublicIPsE()
	if err != nil {
		return fmt.Errorf("load secondary mTLS allowlist sources: %w", err)
	}
	var failures []string
	for _, ip := range ips {
		if err := RestrictAgentMTLSToIP(ip); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", ip, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("sync secondary mTLS allowlist failed: %s", strings.Join(failures, "; "))
	}
	return nil
}
