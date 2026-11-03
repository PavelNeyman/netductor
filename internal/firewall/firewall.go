// Package firewall applies role-based host firewall for netductor nodes.
// Preferred backend: ufw (installed if missing). Fallback: iptables-nft or legacy iptables.
// Never silently skip: Apply returns error if no backend can enforce rules.
package firewall

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/PavelNeyman/netductor/internal/format"

	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/paths"
)

type Backend string

const (
	BackendUFW      Backend = "ufw"
	BackendIPTables Backend = "iptables"
	BackendNone     Backend = "none"
)

// Status is JSON-friendly for doctor / API / UI.
type Status struct {
	Backend    Backend  `json:"backend"`
	Active     bool     `json:"active"`
	Role       string   `json:"role"`
	OK         bool     `json:"ok"`
	Detail     string   `json:"detail,omitempty"`
	OpenWAN    []string `json:"open_wan"`   // expected public allows
	DenyWAN    []string `json:"deny_wan"`   // expected explicit denies
	Restricted []string `json:"restricted"` // CIDR-limited
	Warnings   []string `json:"warnings,omitempty"`
	CheckedAt  string   `json:"checked_at"`
}

type wssExpectation struct {
	Port   string
	Source string
}

type wssPeerIndex struct {
	Peers []struct {
		PublicIP string `json:"public_ip"`
		State    string `json:"state"`
		Slot     struct {
			WSSSP int `json:"wss_sp"`
		} `json:"slot"`
	} `json:"peers"`
}

type wssMaterial struct {
	Primary   string `json:"primary_ip"`
	Secondary string `json:"secondary_ip"`
	WSSSP     int    `json:"wss_sp"`
	WSSPS     int    `json:"wss_ps"`
}

func readFirewallJSON(path string, dst any) (bool, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		if err == unix.ENOENT { return false, nil }
		return false, fmt.Errorf("open %s: %w", path, err)
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil { _ = unix.Close(fd); return false, fmt.Errorf("open file handle for %s", path) }
	defer f.Close()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil { return false, fmt.Errorf("stat %s: %w", path, err) }
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o022 != 0 || st.Size < 0 || st.Size > 1<<20 {
		return false, fmt.Errorf("%s must be an owned regular file, not group/world-writable, and <=1MiB", path)
	}
	// These files can contain WireGuard private keys. Tighten legacy
	// owner-readable modes in place; never accept group/world-writable files.
	if st.Mode&0o077 != 0 {
		if err := f.Chmod(0o600); err != nil { return false, fmt.Errorf("secure permissions on %s: %w", path, err) }
		if err := f.Sync(); err != nil { return false, fmt.Errorf("sync secure permissions on %s: %w", path, err) }
	}
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil { return false, fmt.Errorf("read %s: %w", path, err) }
	if len(b) > 1<<20 { return false, fmt.Errorf("%s exceeds 1MiB", path) }
	if err := json.Unmarshal(b, dst); err != nil { return false, fmt.Errorf("parse %s: %w", path, err) }
	return true, nil
}

func roleWSSPorts(role string) []string {
	if role == "primary" { return []string{"8444", "8454", "8464", "8474"} }
	return []string{"8445", "8455", "8465", "8475"}
}

func wssExpectations(role string) ([]wssExpectation, error) {
	dir := filepath.Join(paths.EtcDir(), "svc-paths")
	var out []wssExpectation
	add := func(port int, source string) error {
		ip := net.ParseIP(strings.TrimSpace(source))
		if port < 1 || port > 65535 || ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid %s WSS peer tuple port=%d source=%q", role, port, source)
		}
		valid := false
		for _, p := range roleWSSPorts(role) { if p == strconv.Itoa(port) { valid = true; break } }
		if !valid { return fmt.Errorf("unexpected %s WSS port %d (not in reserved slot table)", role, port) }
		out = append(out, wssExpectation{Port: strconv.Itoa(port), Source: ip.To4().String()})
		return nil
	}
	if role == "primary" {
		var idx wssPeerIndex
		exists, err := readFirewallJSON(filepath.Join(dir, "peers.json"), &idx)
		if err != nil { return nil, err }
		if exists {
			for _, p := range idx.Peers {
				if p.State == "removed" || p.State == "removing" { continue }
				if err := add(p.Slot.WSSSP, p.PublicIP); err != nil { return nil, err }
			}
			// An existing empty registry is authoritative: do not resurrect a
			// retired legacy material.json peer after the last managed peer is removed.
			return out, nil
		}
		var legacy wssMaterial
		exists, err = readFirewallJSON(filepath.Join(dir, "material.json"), &legacy)
		if err != nil { return nil, err }
		if exists && legacy.Secondary != "" {
			port := legacy.WSSSP
			if port == 0 { port = 8444 }
			if err := add(port, legacy.Secondary); err != nil { return nil, err }
		}
		return out, nil
	}
	var local wssMaterial
	exists, err := readFirewallJSON(filepath.Join(dir, "material.json"), &local)
	if err != nil { return nil, err }
	if exists && local.Primary != "" {
		port := local.WSSPS
		if port == 0 { port = 8445 }
		if err := add(port, local.Primary); err != nil { return nil, err }
	}
	return out, nil
}

// verifyWSSRules enforces per-peer source-restricted WSS. No WSS port is ever
// expected to be globally exposed; prepared peers are included for blue-green.
func verifyWSSRules(rules []ufwRule, role string) error {
	expected, err := wssExpectations(role)
	if err != nil { return err }
	ports := append(roleWSSPorts("primary"), roleWSSPorts("secondary")...)
	for _, e := range expected {
		action, found := firstTCPActionFromSource(rules, e.Port, e.Source, false)
		if !found || action != "ALLOW" {
			return fmt.Errorf("%s WSS %s from %s first action=%q found=%t (want ALLOW)", role, e.Port, e.Source, action, found)
		}
	}
	for _, rule := range rules {
		if !ufwRuleAllowsTraffic(rule.Action) { continue }
		isWSS := false
		for _, port := range ports { if ufwRuleMatchesPortTCP(rule, port) { isWSS = true; break } }
		if !isWSS { continue }
		if rule.IPv6 || ufwFromIsWAN(rule.From) {
			return fmt.Errorf("%s WSS port %s has broad/IPv6 ALLOW from %s", role, rule.Port, rule.From)
		}
		matched := false
		for _, e := range expected {
			if e.Port == rule.Port && ufwSourceCovers(e.Source, rule.From) { matched = true; break }
		}
		if !matched { return fmt.Errorf("%s WSS port %s has unregistered ALLOW from %s", role, rule.Port, rule.From) }
	}
	return nil
}

// EnsureWSSPeerRulesLocked installs source-specific rules for every registered peer
// before callers remove legacy global WSS allows. The caller must hold
// AcquireMutationLock; this deliberately does not acquire it recursively.
func EnsureWSSPeerRulesLocked(role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "primary" && role != "secondary" {
		return fmt.Errorf("WSS peer policy: invalid role %q", role)
	}
	expected, err := wssExpectations(role)
	if err != nil {
		return err
	}
	status, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil {
		return fmt.Errorf("inspect UFW before preserving WSS peers: %v (%s)", err, truncate(string(status), 160))
	}
	rules := parseUfwNumbered(string(status))
	active := strings.Contains(string(status), "Status: active")
	if active {
		numberedLines := 0
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "[") {
				numberedLines++
			}
		}
		if len(rules) != numberedLines {
			return fmt.Errorf("cannot completely parse UFW rules before WSS migration (parsed %d of %d)", len(rules), numberedLines)
		}
	}
	for _, e := range expected {
		// A deny that would become effective after the global ALLOW is removed
		// must be detected before mutation; otherwise migration could strand
		// an already-connected peer.
		if active {
			for _, rule := range rules {
				if rule.IPv6 || !ufwRuleMatchesPortTCP(rule, e.Port) {
					continue
				}
				if (rule.Action == "DENY" || rule.Action == "REJECT") && ufwSourceOverlaps(rule.From, e.Source) {
					return fmt.Errorf("WSS %s from %s is shadowed by existing %s from %s; refusing legacy-rule removal",
						e.Port, e.Source, rule.Action, rule.From)
				}
			}
			if hasExactWSSPeerAllow(rules, e) {
				continue
			}
		}
		comment := "nd-wss-sp"
		if role == "secondary" {
			comment = "nd-wss-ps"
		}
		out, addErr := exec.Command("ufw", "allow", "from", e.Source, "to", "any", "port", e.Port, "proto", "tcp", "comment", comment).CombinedOutput()
		if addErr != nil {
			return fmt.Errorf("preserve WSS %s from %s before legacy migration: %v (%s)", e.Port, e.Source, addErr, truncate(string(out), 160))
		}
		status, err = exec.Command("ufw", "status", "numbered").CombinedOutput()
		if err != nil {
			return fmt.Errorf("verify preserved WSS rule %s: %v (%s)", e.Port, err, truncate(string(status), 160))
		}
		rules = parseUfwNumbered(string(status))
		active = strings.Contains(string(status), "Status: active")
		if active && !hasExactWSSPeerAllow(rules, e) {
			return fmt.Errorf("UFW did not persist exact WSS allow for %s from %s", e.Port, e.Source)
		}
	}
	return nil
}

func hasExactWSSPeerAllow(rules []ufwRule, e wssExpectation) bool {
	target := net.ParseIP(e.Source).To4()
	if target == nil {
		return false
	}
	for _, rule := range rules {
		if rule.IPv6 || rule.Action != "ALLOW" || !ufwRuleMatchesPortTCP(rule, e.Port) {
			continue
		}
		from := strings.TrimSpace(rule.From)
		if ip := net.ParseIP(from); ip != nil && ip.To4() != nil && ip.Equal(target) {
			return true
		}
		ip, network, err := net.ParseCIDR(from)
		if err == nil {
			ones, bits := network.Mask.Size()
			if bits == 32 && ones == 32 && ip.To4() != nil && ip.To4().Equal(target) {
				return true
			}
		}
	}
	return false
}

func ensureWSSAllowRules(role string) error {
	expected, err := wssExpectations(role)
	if err != nil { return err }
	status, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil { return fmt.Errorf("inspect UFW before WSS peer migration: %v (%s)", err, truncate(string(status), 160)) }
	rules := parseUfwNumbered(string(status))
	active := strings.Contains(string(status), "Status: active")
	for _, e := range expected {
		if active {
			action, found := firstTCPActionFromSource(rules, e.Port, e.Source, false)
			if found {
				if action != "ALLOW" { return fmt.Errorf("WSS %s from %s is shadowed by %s; refusing to reorder administrator rules", e.Port, e.Source, action) }
				continue
			}
		}
		comment := "nd-wss-sp"
		if role == "secondary" { comment = "nd-wss-ps" }
		out, addErr := exec.Command("ufw", "allow", "from", e.Source, "to", "any", "port", e.Port, "proto", "tcp", "comment", comment).CombinedOutput()
		if addErr != nil { return fmt.Errorf("install source-restricted WSS %s from %s: %v (%s)", e.Port, e.Source, addErr, truncate(string(out), 160)) }
		status, err = exec.Command("ufw", "status", "numbered").CombinedOutput()
		if err != nil { return fmt.Errorf("verify WSS rule %s: %v (%s)", e.Port, err, truncate(string(status), 160)) }
		rules = parseUfwNumbered(string(status))
		active = strings.Contains(string(status), "Status: active")
	}
	if !active { return nil } // UFW hides ordered rules until enabled; post-enable verification is mandatory.
	return verifyWSSRules(rules, role)
}

// RemoveLegacyGlobalWSSRules migrates old Netductor-wide WSS ALLOW rules. It
// removes only rules carrying Netductor's historical comments; administrator
// rules are preserved and subsequently cause a fail-closed policy check.
func RemoveLegacyGlobalWSSRules(status string) error {
	var numbers []int
	for _, rule := range parseUfwNumbered(status) {
		if rule.Action != "ALLOW" || !ufwFromIsWAN(rule.From) || (rule.Port != "8444" && rule.Port != "8445") { continue }
		if strings.Contains(rule.Raw, "nd-wss-sp") || strings.Contains(rule.Raw, "nd-wss-ps") {
			numbers = append(numbers, rule.Num)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(numbers)))
	for _, n := range numbers {
		out, err := exec.Command("ufw", "--force", "delete", strconv.Itoa(n)).CombinedOutput()
		if err != nil { return fmt.Errorf("remove legacy global WSS rule %d: %v (%s)", n, err, truncate(string(out), 160)) }
	}
	return nil
}

// EnsureInstalled tries to install ufw. Returns backend that will be used.
func EnsureInstalled() (Backend, error) {
	if _, err := exec.LookPath("ufw"); err == nil {
		return BackendUFW, nil
	}
	// UFW supports a debconf preseed that enables the firewall during package
	// installation. Force that answer to false: Netductor must install all
	// management-path rules before its explicit ufw enable step.
	preseed := exec.Command("debconf-set-selections")
	preseed.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	preseed.Stdin = strings.NewReader("ufw ufw/enable boolean false\n")
	if out, err := preseed.CombinedOutput(); err != nil {
		return BackendNone, fmt.Errorf("firewall: disable UFW install-time activation via debconf: %v (%s)", err, truncate(string(out), 200))
	}
	cmd := exec.Command("apt-get", "install", "-y", "ufw")
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, installErr := cmd.CombinedOutput()
	if installErr == nil {
		if _, lookupErr := exec.LookPath("ufw"); lookupErr == nil {
			return BackendUFW, nil
		}
	} else if _, lookupErr := exec.LookPath("ufw"); lookupErr == nil {
		// Collect() prefers UFW whenever its binary exists. Falling back to
		// iptables after a partial package installation would apply one backend
		// while status/health reports the other.
		return BackendNone, fmt.Errorf("firewall: UFW installation failed but its binary is present; refusing an ambiguous iptables fallback: %v (%s)", installErr, truncate(string(out), 200))
	}
	// Fallback only when UFW is genuinely unavailable, so Apply and Collect
	// cannot select different backends based on binary presence.
	if _, lookupErr := exec.LookPath("iptables"); lookupErr == nil {
		fmt.Fprintf(os.Stderr, "firewall: ufw install failed (%v: %s) — using iptables fallback\n", installErr, truncate(string(out), 200))
		return BackendIPTables, nil
	}
	return BackendNone, fmt.Errorf("firewall: cannot install ufw and no iptables: %v %s", installErr, truncate(string(out), 300))
}

func truncate(s string, n int) string { return format.Truncate(strings.TrimSpace(s), n) }


// acquireFirewallApplyLock serializes Netductor firewall mutations across processes.
// It does not coordinate with UFW, iptables, or other external firewall managers.
func AcquireMutationLock() (func(), error) {
	return acquireMutationLock(0)
}

// AcquireMutationLockTimeout acquires the same cross-process lock but refuses
// to wait indefinitely. Bootstrap rollback uses this so recovery cannot hang
// behind another firewall operation forever.
func AcquireMutationLockTimeout(timeout time.Duration) (func(), error) {
	if timeout <= 0 { return nil, fmt.Errorf("firewall: lock timeout must be positive") }
	return acquireMutationLock(timeout)
}

func acquireMutationLock(timeout time.Duration) (func(), error) {
	const lockPath = "/run/lock/netductor-firewall.lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("firewall: create lock directory: %w", err)
	}
	fd, err := unix.Open(lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("firewall: open apply lock: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("firewall: inspect apply lock: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != 0 || st.Mode&0o022 != 0 {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("firewall: apply lock must be a root-owned regular file not writable by group or others")
	}
	if timeout == 0 {
		if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("firewall: acquire apply lock: %w", err)
		}
	} else {
		deadline := time.Now().Add(timeout)
		for {
			err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
			if err == nil { break }
			if err != unix.EWOULDBLOCK && err != unix.EAGAIN {
				_ = unix.Close(fd)
				return nil, fmt.Errorf("firewall: acquire apply lock: %w", err)
			}
			if !time.Now().Before(deadline) {
				_ = unix.Close(fd)
				return nil, fmt.Errorf("firewall: apply lock acquisition timed out after %s", timeout)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return func() {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
	}, nil
}

// ApplyRole installs backend if needed and applies primary or secondary profile.
func ApplyRole(role string) error { return applyRole(role, false) }

// ApplyRoleBootstrap applies the secondary profile while preserving the factory
// SSH port 22 as a temporary management path. RemoteHardenFinalize must remove
// this exception only after key-only access on the hardened port is verified.
func tcpPortListening(port uint16) (bool, error) {
	if ss, err := exec.LookPath("ss"); err == nil {
		if out, runErr := exec.Command(ss, "-H", "-ltn").CombinedOutput(); runErr == nil {
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 4 && strings.HasSuffix(fields[3], ":"+strconv.Itoa(int(port))) {
					return true, nil
				}
			}
			return false, nil
		}
	}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) && path == "/proc/net/tcp6" { continue }
			return false, fmt.Errorf("read TCP listener table %s: %w", path, err)
		}
		for _, line := range strings.Split(string(b), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) < 4 || fields[3] != "0A" { continue } // TCP_LISTEN
			colon := strings.LastIndexByte(fields[1], ':')
			if colon < 0 { continue }
			got, parseErr := strconv.ParseUint(fields[1][colon+1:], 16, 16)
			if parseErr == nil && got == uint64(port) { return true, nil }
		}
	}
	return false, nil
}

func ApplyRoleBootstrap(role string) error {
	if strings.ToLower(strings.TrimSpace(role)) != "secondary" {
		return fmt.Errorf("firewall: bootstrap profile is only valid for secondary")
	}
	return applyRole(role, true)
}

func applyRole(role string, bootstrapSSH bool) error {
	unlock, err := AcquireMutationLock()
	if err != nil {
		return err
	}
	defer unlock()

	if bootstrapSSH {
		// EnsureHostBaseline is also used on already-hardened hosts. Do not
		// reopen factory SSH :22 unless a listener is actually present.
		listening, inspectErr := tcpPortListening(22)
		if inspectErr != nil {
			return fmt.Errorf("firewall: inspect factory SSH listener before secondary bootstrap: %w", inspectErr)
		}
		bootstrapSSH = listening
	}

	role = strings.ToLower(strings.TrimSpace(role))
	if role == "core" {
		role = "primary"
	}
	if role == "" {
		var ok bool
		role, ok = DetectRoleStrict()
		if !ok {
			return fmt.Errorf("firewall: cannot safely determine host role; specify primary or secondary explicitly")
		}
	} else if role != "primary" && role != "secondary" {
		return fmt.Errorf("firewall: invalid role %q (want primary or secondary)", role)
	}
	if err := validateRolePorts(role); err != nil {
		return err
	}
	be, err := EnsureInstalled()
	if err != nil {
		return err
	}
	switch be {
	case BackendUFW:
		return applyUFWMode(role, bootstrapSSH)
	case BackendIPTables:
		return applyIPTablesMode(role, bootstrapSSH)
	default:
		return fmt.Errorf("firewall: no backend")
	}
}

// DetectRole returns primary|secondary for baseline/firewall/doctor.
// Unknown hosts retain the legacy primary fallback; safety-sensitive migrations
// must use DetectRoleStrict instead of treating that fallback as evidence.
func DetectRole() string { return detectRole() }

func detectRole() string {
	if role, ok := DetectRoleStrict(); ok {
		return role
	}
	return "primary"
}

// readPersistedRole accepts only a small, root-owned regular file in a
// trusted directory. O_NOFOLLOW and fstat prevent symlink and inode-type tricks.
func readPersistedRole() (string, bool) {
	const path = "/etc/netductor/role"
	if err := validateRootOwnedDirectory("/etc"); err != nil {
		return "", false
	}
	if err := validateRootOwnedDirectory("/etc/netductor"); err != nil {
		return "", false
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return "", false
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return "", false
	}
	defer file.Close()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil ||
		st.Mode&unix.S_IFMT != unix.S_IFREG ||
		st.Uid != 0 || st.Mode&0o022 != 0 || st.Size < 0 || st.Size > 64 {
		return "", false
	}
	b, err := io.ReadAll(io.LimitReader(file, 65))
	if err != nil || len(b) > 64 {
		return "", false
	}
	switch strings.TrimSpace(strings.ToLower(string(b))) {
	case "secondary":
		return "secondary", true
	case "primary", "core":
		return "primary", true
	default:
		return "", false
	}
}

// DetectRoleStrict identifies a role only when a durable or live role marker exists.
// It returns ok=false rather than guessing primary when the host has no usable markers.
func systemdServiceIsActive(name string) bool {
	st, err := exec.Command("systemctl", "is-active", name).CombinedOutput()
	return err == nil && strings.TrimSpace(string(st)) == "active"
}

func DetectRoleStrict() (string, bool) {
	// Live secondary agent is definitive — prevents wrong role=primary after node upgrade paths.
	if systemdServiceIsActive("netductor-secondary-agent") {
		return "secondary", true
	}
	// Explicit role file (written by install / firewall apply).
	if role, ok := readPersistedRole(); ok {
		return role, true
	}
	// Primary control-plane markers (bundle.json may exist on primary after export — do NOT treat as secondary).
	if isTrustedRootOwnedFile(filepath.Join(paths.EtcDir(), "READY.txt")) {
		return "primary", true
	}
	if systemdServiceIsActive("netductor-api") {
		return "primary", true
	}
	if systemdServiceIsActive("netductor-telegram-bot") {
		return "primary", true
	}
	// Last resort: secondary bundle only if no primary markers above.
	if isTrustedRootOwnedFile(filepath.Join(paths.SecondaryDir(), "bundle.json")) {
		return "secondary", true
	}
	return "", false
}

// persistRole writes /etc/netductor/role so status matches last apply.
// isTrustedRootOwnedFile accepts only a root-owned regular file with a
// non-writable-by-group/other directory ancestry and no symlink components.
func isTrustedRootOwnedFile(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	for dir := filepath.Clean(filepath.Dir(path)); ; dir = filepath.Dir(dir) {
		if err := validateRootOwnedDirectory(dir); err != nil {
			return false
		}
		if dir == "/" {
			break
		}
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return false
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	return unix.Fstat(fd, &st) == nil &&
		st.Mode&unix.S_IFMT == unix.S_IFREG &&
		st.Uid == 0 &&
		st.Mode&0o022 == 0
}

func validateRootOwnedDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a real directory (symlinks are not accepted)", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return fmt.Errorf("%s is not owned by root", path)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%s is group- or world-writable", path)
	}
	return nil
}

// persistRole writes /etc/netductor/role so status matches last apply.
func persistRole(role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "primary" && role != "secondary" {
		return fmt.Errorf("firewall: refusing to persist invalid role %q", role)
	}
	const dir = "/etc/netductor"
	if err := validateRootOwnedDirectory("/etc"); err != nil {
		return fmt.Errorf("firewall: unsafe parent for role directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("firewall: create role directory: %w", err)
	}
	if err := validateRootOwnedDirectory(dir); err != nil {
		return fmt.Errorf("firewall: unsafe role directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".role-*")
	if err != nil {
		return fmt.Errorf("firewall: create role temp file: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return fmt.Errorf("firewall: chmod role temp file: %w", err)
	}
	if _, err := f.WriteString(role+"\n"); err != nil {
		_ = f.Close()
		return fmt.Errorf("firewall: write role: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("firewall: sync role: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("firewall: close role: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(dir, "role")); err != nil {
		return fmt.Errorf("firewall: replace role: %w", err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("firewall: open role directory for sync: %w", err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("firewall: sync role directory: %w", err)
	}
	return nil
}


// validateRolePorts rejects invalid or colliding ports before installing or
// mutating a firewall backend. In particular, SSH must never share a port
// with a service that this role explicitly denies or source-restricts.
func validateRolePorts(role string) error {
	parse := func(label, value string, optional bool) (int, error) {
		value = strings.TrimSpace(value)
		if optional && (value == "" || value == "0") {
			return 0, nil
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 65535 {
			return 0, fmt.Errorf("firewall: invalid %s port %q (want numeric 1..65535)", label, value)
		}
		return n, nil
	}
	ssh, err := parse("SSH", sshPort(), false)
	if err != nil {
		return err
	}
	reserved := map[int]string{22: "legacy SSH", 443: "VLESS"}
	add := func(label, value string, optional bool) error {
		n, err := parse(label, value, optional)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if previous, exists := reserved[n]; exists {
			return fmt.Errorf("firewall: %s port %d conflicts with %s", label, n, previous)
		}
		reserved[n] = label
		return nil
	}
	if role == "primary" {
		for _, item := range []struct{ label, port string }{
			{"redirect HTTPS", ndconfig.RedirectHTTPSPort()},
			{"agent mTLS", ndconfig.AgentMTLSPort()},
		} {
			if err := add(item.label, item.port, false); err != nil {
				return err
			}
		}
		if err := add("Lampac", ndconfig.LampacPort(), true); err != nil {
			return err
		}
		for _, item := range []struct{ label, port string }{
			{"HTTP", "80"}, {"primary WSS", "8444"}, {"plain agent", "8788"}, {"service uplink", "9443"},
		} {
			if err := add(item.label, item.port, false); err != nil {
				return err
			}
		}
	} else {
		for _, item := range []struct{ label, port string }{
			{"secondary WSS", "8445"}, {"secondary API", "8787"}, {"plain agent", "8788"}, {"HTTP", "80"}, {"Zabbix agent", "10050"}, {"Zabbix server", "10051"},
		} {
			if err := add(item.label, item.port, false); err != nil {
				return err
			}
		}
	}
	// Ports owned by the opposite role must remain blocked, but their
	// configured values must never collide with this role's active services.
	checkInactive := func(label, value string, optional bool) error {
		n, err := parse(label, value, optional)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if n == ssh {
			return fmt.Errorf("firewall: inactive %s port %d conflicts with SSH port", label, n)
		}
		if active, exists := reserved[n]; exists {
			return fmt.Errorf("firewall: inactive %s port %d conflicts with active %s port", label, n, active)
		}
		return nil
	}
	if role == "primary" {
		for _, item := range []struct{ label, port string }{
			{"secondary WSS", "8445"}, {"secondary API", "8787"},
		} {
			if err := checkInactive(item.label, item.port, false); err != nil {
				return err
			}
		}
	} else {
		for _, item := range []struct{ label, port string }{
			{"primary WSS", "8444"}, {"primary service uplink", "9443"},
			{"primary mTLS", ndconfig.AgentMTLSPort()}, {"primary redirect HTTPS", ndconfig.RedirectHTTPSPort()},
		} {
			if err := checkInactive(item.label, item.port, false); err != nil {
				return err
			}
		}
		if err := checkInactive("primary Lampac", ndconfig.LampacPort(), true); err != nil {
			return err
		}
	}
	if previous, exists := reserved[ssh]; exists {
		return fmt.Errorf("firewall: SSH port %d conflicts with reserved %s port", ssh, previous)
	}
	return nil
}

func sshPort() string {
	return strconv.Itoa(hardening.SSHPort())
}

func ufwIPv6Enabled() (bool, error) {
	b, err := os.ReadFile("/etc/default/ufw")
	if err != nil {
		return false, fmt.Errorf("read /etc/default/ufw: %w", err)
	}
	found := false
	enabled := false
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "IPV6" {
			continue
		}
		if found {
			return false, fmt.Errorf("duplicate IPV6 setting in /etc/default/ufw")
		}
		found = true
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		enabled = strings.EqualFold(value, "yes")
	}
	if !found {
		return false, fmt.Errorf("IPV6 setting is missing from /etc/default/ufw")
	}
	if !enabled {
		return false, fmt.Errorf("IPV6 is not enabled in /etc/default/ufw")
	}
	return true, nil
}

// ufwPreflightPolicy rejects existing ordered rules that would override or
// broaden the role policy. Missing rules are safe to append; conflicting rules
// are never deleted automatically because they may be administrator-owned.
func ufwPreflightPolicy(status, role string) error {
	return ufwPreflightPolicyMode(status, role, false)
}

func ufwPreflightPolicyMode(status, role string, bootstrapSSH bool) error {
	rules := parseUfwNumbered(status)
	numberedLines := 0
	for _, line := range strings.Split(status, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			numberedLines++
		}
	}
	if len(rules) != numberedLines {
		return fmt.Errorf("cannot completely parse existing UFW rules (parsed %d of %d)", len(rules), numberedLines)
	}
	var conflicts []string
	checkWAN := func(port, want, label string) {
		for _, ipv6 := range []bool{false, true} {
			action, found := firstTCPActionFromWANFamily(rules, port, ipv6)
			if found && action != want {
				family := "IPv4"
				if ipv6 { family = "IPv6" }
				conflicts = append(conflicts, fmt.Sprintf("%s %s first %s WAN action is %s, want %s", label, port, family, action, want))
			}
		}
	}
	checkNoAllow := func(port, label string) {
		for _, rule := range rules {
			if ufwRuleMatchesPortTCP(rule, port) && ufwRuleAllowsTraffic(rule.Action) {
				family := "IPv4"
				if rule.IPv6 { family = "IPv6" }
				conflicts = append(conflicts, fmt.Sprintf("%s %s has existing %s ALLOW from %s", label, port, family, rule.From))
			}
		}
	}
	checkNoDeny := func(port, label string) {
		for _, rule := range rules {
			if ufwRuleMatchesPortTCP(rule, port) && (rule.Action == "DENY" || rule.Action == "REJECT") {
				conflicts = append(conflicts, fmt.Sprintf("%s %s has existing %s %s from %s", label, port, rule.Action, rule.Proto, rule.From))
			}
		}
	}
	checkSource := func(port, source, label string) {
		action, found := firstTCPActionFromSource(rules, port, source, false)
		if found && action != "ALLOW" {
			conflicts = append(conflicts, fmt.Sprintf("%s %s first rule for %s is %s, want ALLOW", label, port, source, action))
		}
	}
	for _, item := range []struct{ port, action, label string }{
		{"443", "ALLOW", "vless"}, {"10050", "DENY", "zabbix"}, {"10051", "DENY", "zabbix"},
	} {
		checkWAN(item.port, item.action, item.label)
	}
	if !bootstrapSSH {
		checkWAN("22", "DENY", "ssh22")
	}
	for _, item := range []struct{ port, label string }{
		{"443", "vless"},
	} {
		checkNoDeny(item.port, item.label)
	}
	// A source-specific SSH DENY/REJECT can precede a global SSH ALLOW.
	// The WAN-only matcher below intentionally ignores such rules, so reject
	// them explicitly rather than claiming the management path is preserved.
	for _, rule := range rules {
		if !bootstrapSSH && ufwRuleMatchesPortTCP(rule, sshPort()) && (rule.Action == "DENY" || rule.Action == "REJECT") {
			conflicts = append(conflicts, fmt.Sprintf("ssh %s has an existing %s from %s", sshPort(), rule.Action, rule.From))
		}
	}
	for _, item := range []struct{ port, label string }{
		{"10050", "zabbix"}, {"10051", "zabbix"},
	} {
		checkNoAllow(item.port, item.label)
	}
	if !bootstrapSSH {
		for _, rule := range ufwAllowsBeforeFirstWANAction(rules, "22", "DENY") {
			conflicts = append(conflicts, fmt.Sprintf("ssh22 has an effective %s ALLOW from %s before the WAN deny", rule.Proto, rule.From))
		}
	}
	if role == "primary" {
		for _, item := range []struct{ port, action, label string }{
			{ndconfig.RedirectHTTPSPort(), "ALLOW", "redirect"},
			{"80", "ALLOW", "http-acme"},
			{"8787", "DENY", "secondary-api"},
			{"8788", "DENY", "plain-agent"}, {"9443", "DENY", "service-uplink"},
		} { checkWAN(item.port, item.action, item.label) }
		for _, item := range []struct{ port, label string }{
			{ndconfig.RedirectHTTPSPort(), "redirect"}, {"80", "http-acme"},
		} { checkNoDeny(item.port, item.label) }
		for _, item := range []struct{ port, label string }{
			{"8787", "secondary-api"}, {"8788", "plain-agent"},
		} { checkNoAllow(item.port, item.label) }
		uplink := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()}
		for _, source := range uplink { checkSource("9443", source, "service-uplink") }
		mtls := ndconfig.AgentMTLSPort()
		trustedMTLS := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()}
		for _, source := range trustedMTLS { checkSource(mtls, source, "mtls") }
		for _, rule := range rules {
			if !ufwRuleMatchesPortTCP(rule, "9443") || !ufwRuleAllowsTraffic(rule.Action) { continue }
			trusted := false
			if !rule.IPv6 {
				for _, source := range uplink {
					if ufwSourceContainedBy(rule.From, source) { trusted = true; break }
				}
			}
			if !trusted { conflicts = append(conflicts, "service-uplink has an existing untrusted ALLOW from "+rule.From) }
		}
		for _, rule := range rules {
			if !ufwRuleMatchesPortTCP(rule, mtls) || !ufwRuleAllowsTraffic(rule.Action) { continue }
			trusted := false
			if !rule.IPv6 {
				for _, source := range trustedMTLS {
					if ufwSourceContainedBy(rule.From, source) { trusted = true; break }
				}
			}
			if !trusted { conflicts = append(conflicts, "mtls has an existing untrusted ALLOW from "+rule.From) }
		}
		lp := ndconfig.LampacPort()
		if lp != "" && lp != "0" {
			checkWAN(lp, "DENY", "lampac")
			for _, rule := range rules {
				if ufwRuleMatchesPortTCP(rule, lp) && ufwRuleAllowsTraffic(rule.Action) {
					conflicts = append(conflicts, "lampac has an existing source-based ALLOW from "+rule.From)
				}
			}
		}
	} else {
		for _, port := range []string{"80", "8787", "8788", "8789"} {
			checkWAN(port, "DENY", "secondary-restricted")
		}
		for _, port := range []string{"80", "8787", "8788", "8789"} {
			checkNoAllow(port, "secondary-restricted")
		}
		// A previous primary profile may have left source-restricted allows
		// behind. UFW does not remove them when applying the secondary role.
		primaryOnly := []struct{ port, label string }{
			{"9443", "primary-service-uplink"},
			{ndconfig.AgentMTLSPort(), "primary-mtls"},
			{ndconfig.RedirectHTTPSPort(), "primary-redirect"},
		}
		if lp := ndconfig.LampacPort(); lp != "" && lp != "0" {
			primaryOnly = append(primaryOnly, struct{ port, label string }{lp, "primary-lampac"})
		}
		for _, item := range primaryOnly {
			for _, rule := range rules {
				if ufwRuleMatchesPortProtocol(rule, item.port, "") && ufwRuleAllowsTraffic(rule.Action) {
					conflicts = append(conflicts, fmt.Sprintf("%s %s has an existing %s ALLOW from %s", item.label, item.port, rule.Proto, rule.From))
				}
			}
		}
	}
	if strings.Contains(status, "Status: active") {
		if err := verifyWSSRules(rules, role); err != nil {
			conflicts = append(conflicts, "WSS peer policy: "+err.Error())
		}
	}
	if len(conflicts) != 0 {
		return fmt.Errorf("existing UFW policy conflicts; refusing mutation: %s", strings.Join(conflicts, "; "))
	}
	return nil
}

func applyUFW(role string) error { return applyUFWMode(role, false) }

// preflightSecondaryRestrictedAllows detects pre-existing administrator ALLOWs
// before WSS migration can mutate any numbered UFW rules. UFW is first-match;
// appending a DENY later does not neutralize an earlier ALLOW.
func preflightSecondaryRestrictedAllows(status string) error {
	rules := parseUfwNumbered(status)
	numbered := 0
	for _, line := range strings.Split(status, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "[") { numbered++ }
	}
	if len(rules) != numbered {
		return fmt.Errorf("cannot completely parse UFW rules before secondary migration (parsed %d of %d)", len(rules), numbered)
	}
	for _, port := range []string{"80", "8787", "8788", "8789"} {
		for _, rule := range rules {
			if ufwRuleMatchesPortProtocol(rule, port, "tcp") && ufwRuleAllowsTraffic(rule.Action) {
				return fmt.Errorf("secondary restricted port %s has pre-existing %s ALLOW from %s; resolve administrator rule explicitly before applying role", port, rule.Proto, rule.From)
			}
		}
	}
	return nil
}

func applyUFWMode(role string, bootstrapSSH bool) error {
	if _, err := ufwIPv6Enabled(); err != nil {
		return fmt.Errorf("ufw IPv6 policy prerequisite: %w", err)
	}
	run := func(args ...string) error {
		cmd := exec.Command("ufw", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ufw %v: %v (%s)", args, err, truncate(string(out), 160))
		}
		return nil
	}
	must := func(args ...string) error { return run(args...) }
	if os.Getenv("NETDUCTOR_FW_RESET") == "1" {
		return fmt.Errorf("NETDUCTOR_FW_RESET is disabled: resetting an active firewall is not transaction-safe; remove the variable and apply the role policy without reset")
	}
	sp := sshPort()
	// Preserve the active management path before tightening the default policy.
	// Avoid adding a duplicate on every apply when the first WAN rule already
	// permits SSH for that address family.
	current, inspectErr := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if inspectErr != nil {
		return fmt.Errorf("inspect UFW rules before SSH safeguard: %v (%s)", inspectErr, truncate(string(current), 200))
	}
	// Known factory/admin 80/tcp ALLOW (and other restricted-port allows)
	// must stop before source-WSS migration changes the firewall.
	if role == "secondary" {
		if err := preflightSecondaryRestrictedAllows(string(current)); err != nil {
			return fmt.Errorf("UFW secondary preflight: %w", err)
		}
	}
	// Validate the complete peer inventory before any firewall mutation.
	// Then migrate only the exact historical Netductor global WSS rules.
	if _, err := wssExpectations(role); err != nil {
		return fmt.Errorf("read WSS peer inventory before firewall mutation: %w", err)
	}
	// Install exact peer allows first. Removing a legacy global WSS rule
	// before these exist can disconnect an already-running secondary.
	if err := EnsureWSSPeerRulesLocked(role); err != nil {
		return fmt.Errorf("preserve WSS peer access before migration: %w", err)
	}
	// Adding source-restricted rules changes UFW rule numbers. Refresh before
	// deleting by number so we never remove a shifted administrator rule.
	current, inspectErr = exec.Command("ufw", "status", "numbered").CombinedOutput()
	if inspectErr != nil {
		return fmt.Errorf("re-inspect UFW rules before legacy WSS removal: %v (%s)", inspectErr, truncate(string(current), 200))
	}
	if err := RemoveLegacyGlobalWSSRules(string(current)); err != nil {
		return err
	}
	if err := ensureWSSAllowRules(role); err != nil {
		return err
	}
	current, inspectErr = exec.Command("ufw", "status", "numbered").CombinedOutput()
	if inspectErr != nil {
		return fmt.Errorf("inspect UFW rules after WSS migration: %v (%s)", inspectErr, truncate(string(current), 200))
	}
	if err := ufwPreflightPolicyMode(string(current), role, bootstrapSSH); err != nil {
		return fmt.Errorf("UFW preflight: %w", err)
	}
	if strings.Contains(string(current), "Status: active") {
		rules := parseUfwNumbered(string(current))
		numberedLines := 0
		for _, line := range strings.Split(string(current), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "[") {
				numberedLines++
			}
		}
		if len(rules) != numberedLines {
			return fmt.Errorf("inspect UFW rules before mutation: cannot completely parse numbered rules (parsed %d of %d)", len(rules), numberedLines)
		}
		needSSHInsert := false
		for _, ipv6 := range []bool{false, true} {
			action, found := firstTCPActionFromWANFamily(rules, sp, ipv6)
			if !found || action != "ALLOW" {
				needSSHInsert = true
				break
			}
		}
		if needSSHInsert {
			if err := must("insert", "1", "allow", sp+"/tcp", "comment", "netductor-ssh"); err != nil {
				return fmt.Errorf("install SSH allow rule before default deny: %w", err)
			}
		}
	} else {
		// Inactive UFW does not enforce the listed rules. Insert the safeguard
		// before changing defaults and enabling the firewall.
		if err := must("insert", "1", "allow", sp+"/tcp", "comment", "netductor-ssh"); err != nil {
			return fmt.Errorf("install SSH allow rule before default deny: %w", err)
		}
	}
	if bootstrapSSH {
		// Keep factory SSH reachable while the new key-only listener is prepared.
		// Reuse an already-effective first rule; otherwise insert our tagged rule.
		rules := parseUfwNumbered(string(current))
		ready := strings.Contains(string(current), "netductor-ssh-bootstrap")
		for _, ipv6 := range []bool{false, true} {
			action, found := firstTCPActionFromWANFamily(rules, "22", ipv6)
			if !found || action != "ALLOW" { ready = false; break }
		}
		if !ready {
			if err := must("insert", "1", "allow", "22/tcp", "comment", "netductor-ssh-bootstrap"); err != nil {
				return fmt.Errorf("install temporary factory SSH allow before default deny: %w", err)
			}
		}
	}

	// H-2: critical policy steps must not be ignored. Only tighten incoming
	// traffic after the configured SSH port has an explicit allow rule.
	if err := must("default", "deny", "incoming"); err != nil {
		return err
	}
	if err := must("default", "allow", "outgoing"); err != nil {
		return err
	}
	if err := must("default", "deny", "routed"); err != nil { return fmt.Errorf("set default routed deny: %w", err) }
	if err := must("deny", "22/tcp", "comment", "netductor-no-ssh22"); err != nil { return fmt.Errorf("deny SSH port 22: %w", err) }
	if err := must("allow", "443/tcp", "comment", "netductor-vless"); err != nil {
		return err
	}
	if err := must("deny", "10050/tcp", "comment", "no-zabbix"); err != nil { return fmt.Errorf("deny zabbix 10050: %w", err) }
	if err := must("deny", "10051/tcp", "comment", "no-zabbix"); err != nil { return fmt.Errorf("deny zabbix 10051: %w", err) }

	if role == "primary" {
		redir := ndconfig.RedirectHTTPSPort()
		if err := must("allow", redir+"/tcp", "comment", "netductor-redirect"); err != nil {
			return err
		}
		if err := must("allow", redir+"/udp", "comment", "netductor-redirect"); err != nil { return fmt.Errorf("allow redirect UDP: %w", err) }
		if err := must("allow", "80/tcp", "comment", "netductor-http-acme"); err != nil {
			return err
		}
		if err := must("deny", "8787/tcp", "comment", "no-secondary-api"); err != nil {
			return fmt.Errorf("deny secondary API port: %w", err)
		}
		if err := must("deny", "8445/tcp", "comment", "no-secondary-wss"); err != nil {
			return fmt.Errorf("deny secondary WSS port: %w", err)
		}
		if err := must("deny", "8788/tcp", "comment", "no-plain-agent"); err != nil {
			return err
		}
		mtls := ndconfig.AgentMTLSPort()
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
			if err := must("allow", "from", c, "to", "any", "port", mtls, "proto", "tcp", "comment", "netductor-api"); err != nil {
				return err
			}
		}
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
			if err := must("allow", "from", c, "to", "any", "port", "9443", "proto", "tcp", "comment", "netductor-svc-uplink"); err != nil {
				return err
			}
		}
		if err := must("deny", "9443/tcp", "comment", "no-svc-uplink-wan"); err != nil {
			return err
		}
		lp := ndconfig.LampacPort()
		if lp != "" && lp != "0" {
			if err := must("deny", lp+"/tcp", "comment", "lampac-wan-deny"); err != nil { return fmt.Errorf("deny Lampac WAN port: %w", err) }
		}
	} else {
		if err := must("deny", "8787/tcp", "comment", "no-api-wan"); err != nil {
			return err
		}
		if err := must("deny", "8788/tcp", "comment", "no-plain-agent"); err != nil {
			return err
		}
		if err := must("deny", "8789/tcp", "comment", "no-mtls-wan-secondary"); err != nil {
			return err
		}
		if err := must("deny", "80/tcp", "comment", "no-http-secondary"); err != nil { return fmt.Errorf("deny secondary HTTP port: %w", err) }
	}

	out, err := exec.Command("ufw", "--force", "enable").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw enable: %v (%s)", err, truncate(string(out), 200))
	}

	// Verify the effective policy, not merely the success exit status of each
	// ufw command. Existing earlier rules may override newly appended rules.
	status, statusErr := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if statusErr != nil {
		return fmt.Errorf("ufw post-apply status: %v (%s)", statusErr, truncate(string(status), 200))
	}
	// UFW may hide saved rules while inactive. Re-run the exact-comment legacy
	// migration after activation so old global WSS ALLOW rules cannot reappear.
	if err := RemoveLegacyGlobalWSSRules(string(status)); err != nil { return err }
	if err := ensureWSSAllowRules(role); err != nil { return fmt.Errorf("post-enable WSS peer policy: %w", err) }
	status, statusErr = exec.Command("ufw", "status", "numbered").CombinedOutput()
	if statusErr != nil { return fmt.Errorf("ufw post-migration status: %v (%s)", statusErr, truncate(string(status), 200)) }
	if !strings.Contains(string(status), "Status: active") {
		return fmt.Errorf("ufw post-apply verification failed: firewall is not active (%s)", truncate(string(status), 200))
	}
	verbose, verboseErr := exec.Command("ufw", "status", "verbose").CombinedOutput()
	if verboseErr != nil {
		return fmt.Errorf("ufw post-apply verbose status: %v (%s)", verboseErr, truncate(string(verbose), 200))
	}
	if ok, reason := ufwDefaultIncomingDenied(string(verbose)); !ok {
		return fmt.Errorf("ufw post-apply default policy verification failed: %s", reason)
	}
	if bootstrapSSH {
		rules := parseUfwNumbered(string(status))
		for _, ipv6 := range []bool{false, true} {
			action, found := firstTCPActionFromWANFamily(rules, "22", ipv6)
			if !found || action != "ALLOW" {
				return fmt.Errorf("ufw bootstrap SSH verification failed: port 22 family ipv6=%t action=%q found=%t", ipv6, action, found)
			}
			action, found = firstTCPActionFromWANFamily(rules, sp, ipv6)
			if !found || action != "ALLOW" {
				return fmt.Errorf("ufw bootstrap SSH verification failed: configured port %s family ipv6=%t action=%q found=%t", sp, ipv6, action, found)
			}
		}
	} else if ok, warnings := ufwEffectivePolicy(string(status), role); !ok {
		return fmt.Errorf("ufw post-apply policy verification failed: %s", strings.Join(warnings, "; "))
	}
	if err := persistRole(role); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "firewall: ufw active role=%s ssh=%s\n", role, sp)
	return nil
}

// managedInputJump reports the position of the single Netductor-owned INPUT jump.
// Position is the 1-based rule number accepted by iptables -R, not the line number.
func managedInputJump(rules string) (position int, chain string, err error) {
	inputPosition := 0
	for _, line := range strings.Split(rules, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "-A" || fields[1] != "INPUT" {
			continue
		}
		inputPosition++
		for i := 2; i+1 < len(fields); i++ {
			if fields[i] != "-j" && fields[i] != "--jump" &&
				fields[i] != "-g" && fields[i] != "--goto" {
				continue
			}
			target := fields[i+1]
			if target != "NETDUCTOR" && target != "NETDUCTOR_A" && target != "NETDUCTOR_B" {
				continue
			}
			// Replacing a conditional jump with an unconditional one would
			// silently broaden its scope. A goto also has different return
			// semantics from a jump. Only accept the exact rule shape that this
			// function itself installs; fail closed on legacy/foreign variants.
			if i != 2 || len(fields) != 4 ||
				(fields[2] != "-j" && fields[2] != "--jump") {
				return 0, "", fmt.Errorf("managed INPUT jump to %s is conditional or has unsupported semantics: %s", target, line)
			}
			if chain != "" {
				return 0, "", fmt.Errorf("multiple Netductor INPUT jumps (%s and %s)", chain, target)
			}
			chain = target
			position = inputPosition
			// Continue scanning to detect duplicate managed jumps.
			break
		}
	}
	// A non-empty INPUT chain without a Netductor jump is not an existing
	// managed jump. Returning its rule count here would make the caller replace
	// an unrelated administrator rule instead of inserting our first jump.
	if chain == "" {
		return 0, "", nil
	}
	return position, chain, nil
}

// chainIsReferenced checks all filter-table rules for jumps or gotos to target.
func chainIsReferenced(rules, target string) bool {
	for _, line := range strings.Split(rules, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "-A" {
			continue
		}
		for i := 2; i+1 < len(fields); i++ {
			if (fields[i] == "-j" || fields[i] == "--jump" || fields[i] == "-g" || fields[i] == "--goto") && fields[i+1] == target {
				return true
			}
		}
	}
	return false
}

// requireIPv6Disabled prevents the IPv4-only fallback from reporting success
// while an IPv6 stack could still accept inbound traffic outside its policy.
func requireIPv6Disabled() error {
	const confDir = "/proc/sys/net/ipv6/conf"
	entries, err := os.ReadDir(confDir)
	if err != nil {
		return fmt.Errorf("cannot verify IPv6 state: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(confDir, entry.Name(), "disable_ipv6")
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("cannot verify IPv6 state at %s: %w", path, readErr)
		}
		if strings.TrimSpace(string(b)) != "1" {
			return fmt.Errorf("IPv6 is enabled for %s; refusing IPv4-only iptables fallback", entry.Name())
		}
	}
	return nil
}

// inputStartsWithSSHAllow reports whether the first INPUT rule is an
// unconditional TCP destination-port ACCEPT for the configured SSH port.
// A matching rule later in INPUT is insufficient because earlier rules win.
func inputStartsWithSSHAllow(rules, port string) bool {
	for _, line := range strings.Split(rules, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "-A" || fields[1] != "INPUT" {
			continue
		}
		protocol, destinationPort, target := false, false, false
		for i := 2; i < len(fields); i++ {
			switch fields[i] {
			case "-p", "--protocol":
				if i+1 >= len(fields) || fields[i+1] != "tcp" || protocol {
					return false
				}
				protocol = true
				i++
			case "--dport", "--destination-port":
				if i+1 >= len(fields) || fields[i+1] != port || destinationPort {
					return false
				}
				destinationPort = true
				i++
			case "-j", "--jump":
				if i+1 >= len(fields) || fields[i+1] != "ACCEPT" || target {
					return false
				}
				target = true
				i++
			case "-m":
				// iptables -S may render the implicit TCP matcher explicitly.
				if i+1 >= len(fields) || fields[i+1] != "tcp" {
					return false
				}
				i++
			default:
				// Source/interface/conntrack or any other qualifier would make
				// this less than an unconditional management-path safeguard.
				return false
			}
		}
		return protocol && destinationPort && target
	}
	return false
}

// iptablesHasExactRule checks for one complete rule, not independent option
// fragments. Only the optional implicit TCP matcher emitted by iptables -S
// is tolerated; any additional match changes the rule's security semantics.
func iptablesHasExactRule(rules, chain string, variants ...[]string) bool {
	for _, line := range strings.Split(rules, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "-A" || fields[1] != chain {
			continue
		}
		actual := fields[2:]
		for _, expected := range variants {
			if len(actual) != len(expected) {
				continue
			}
			match := true
			for i := range actual {
				if actual[i] != expected[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func applyIPTables(role string) error { return applyIPTablesMode(role, false) }

func applyIPTablesMode(role string, bootstrapSSH bool) error {
	if err := requireIPv6Disabled(); err != nil {
		return fmt.Errorf("iptables fallback IPv6 prerequisite: %w", err)
	}
	// Build a complete inactive generation, then switch INPUT's single jump.
	// The live chain is never flushed while packets can still traverse it.
	run := func(args ...string) error {
		cmd := exec.Command("iptables", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("iptables %v: %v (%s)", args, err, truncate(string(out), 160))
		}
		return nil
	}
	must := func(args ...string) error { return run(args...) }
	sp := sshPort()
	if _, err := exec.LookPath("netfilter-persistent"); err != nil {
		return fmt.Errorf("iptables fallback: netfilter-persistent is unavailable; active rules cannot be guaranteed across reboot")
	}

	// Keep an unconditional SSH allow as the first INPUT rule. An identical
	// rule found later in the chain is not a safeguard: an earlier DROP/REJECT
	// would still block SSH. The leading rule is intentionally persistent so
	// removing a temporary exception cannot lock out the host after the switch.
	inputBefore, inspectBeforeErr := exec.Command("iptables", "-S", "INPUT").CombinedOutput()
	if inspectBeforeErr != nil {
		return fmt.Errorf("iptables fallback: inspect INPUT before SSH safeguard: %v (%s)", inspectBeforeErr, truncate(string(inputBefore), 200))
	}
	if !inputStartsWithSSHAllow(string(inputBefore), sp) {
		if err := must("-I", "INPUT", "1", "-p", "tcp", "--dport", sp, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("iptables fallback: install leading SSH safeguard: %w", err)
		}
	}
	if bootstrapSSH {
		// Add a uniquely tagged exception after the configured-port safeguard
		// and before the managed-chain jump. Finalize removes only this exact rule.
		args := []string{"-p", "tcp", "--dport", "22", "-m", "comment", "--comment", "netductor-ssh-bootstrap", "-j", "ACCEPT"}
		if err := exec.Command("iptables", append([]string{"-C", "INPUT"}, args...)...).Run(); err != nil {
			if err := must(append([]string{"-I", "INPUT", "2"}, args...)...); err != nil {
				return fmt.Errorf("iptables fallback: install temporary factory SSH safeguard: %w", err)
			}
		}
	}

	inputRules, err := exec.Command("iptables", "-S", "INPUT").CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables fallback: inspect INPUT before chain switch: %v (%s)", err, truncate(string(inputRules), 200))
	}
	jumpPos, activeChain, err := managedInputJump(string(inputRules))
	if err != nil {
		return fmt.Errorf("iptables fallback: unsafe INPUT layout: %w", err)
	}
	stagingChain := "NETDUCTOR_A"
	switch activeChain {
	case "NETDUCTOR_A":
		stagingChain = "NETDUCTOR_B"
	case "NETDUCTOR_B":
		stagingChain = "NETDUCTOR_A"
	case "", "NETDUCTOR":
		stagingChain = "NETDUCTOR_A"
	default:
		return fmt.Errorf("iptables fallback: unsupported active chain %q", activeChain)
	}

	// Refuse to flush a staging chain if any existing filter rule references it.
	// This prevents breaking administrator-created chains that happen to reuse
	// one of the reserved Netductor generation names.
	allRules, inspectErr := exec.Command("iptables", "-S").CombinedOutput()
	if inspectErr != nil {
		return fmt.Errorf("iptables fallback: inspect filter table references: %v (%s)", inspectErr, truncate(string(allRules), 200))
	}
	if chainIsReferenced(string(allRules), stagingChain) {
		return fmt.Errorf("iptables fallback: refusing to rebuild referenced staging chain %s", stagingChain)
	}

	// Create the staging chain if absent; then flush only that inactive chain.
	if err := exec.Command("iptables", "-S", stagingChain).Run(); err != nil {
		if createErr := must("-N", stagingChain); createErr != nil {
			return fmt.Errorf("iptables fallback: create staging chain %s: %w", stagingChain, createErr)
		}
	}
	if err := must("-F", stagingChain); err != nil {
		return fmt.Errorf("iptables fallback: flush inactive chain %s: %w", stagingChain, err)
	}
	allowTCP := func(port string) error {
		return must("-A", stagingChain, "-p", "tcp", "--dport", port, "-j", "ACCEPT")
	}
	denyTCP := func(port string) error {
		return must("-A", stagingChain, "-p", "tcp", "--dport", port, "-j", "DROP")
	}
	allowTCPFrom := func(port, source string) error {
		return must("-A", stagingChain, "-p", "tcp", "-s", source, "--dport", port, "-j", "ACCEPT")
	}
	for _, step := range []struct {
		fn   func() error
		what string
	}{
		{func() error { return allowTCP(sp) }, "allow ssh"},
		{func() error { return denyTCP("22") }, "deny 22"},
		{func() error { return allowTCP("443") }, "allow 443"},
		{func() error { return denyTCP("10050") }, "deny zabbix"},
		{func() error { return denyTCP("10051") }, "deny zabbix"},
	} {
		if err := step.fn(); err != nil {
			return fmt.Errorf("iptables fallback %s: %w", step.what, err)
		}
	}
	// WSS listeners are per-peer and must never be exposed globally. Install
	// source-specific allows first, then drop every reserved WSS port.
	wssPeers, wssErr := wssExpectations(role)
	if wssErr != nil { return fmt.Errorf("iptables fallback WSS inventory: %w", wssErr) }
	for _, peer := range wssPeers {
		if err := allowTCPFrom(peer.Port, peer.Source); err != nil {
			return fmt.Errorf("iptables fallback WSS allow %s from %s: %w", peer.Port, peer.Source, err)
		}
	}
	for _, port := range append(roleWSSPorts("primary"), roleWSSPorts("secondary")...) {
		if err := denyTCP(port); err != nil { return fmt.Errorf("iptables fallback WSS drop %s: %w", port, err) }
	}
	if role == "primary" {
		redir := ndconfig.RedirectHTTPSPort()
		for _, port := range []string{redir, "80"} {
			if err := allowTCP(port); err != nil {
				return fmt.Errorf("iptables fallback allow %s: %w", port, err)
			}
		}
		// UFW exposes the primary redirect listener over both TCP and UDP.
		// Keep the fallback's protocol policy equivalent.
		if err := must("-A", stagingChain, "-p", "udp", "--dport", redir, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("iptables fallback allow redirect UDP %s: %w", redir, err)
		}
		for _, port := range []string{"8787", "8788"} {
			if err := denyTCP(port); err != nil {
				return fmt.Errorf("iptables fallback deny secondary-only port %s: %w", port, err)
			}
		}
		// Match the primary UFW service-uplink policy: only the configured
		// service peers may reach 9443/tcp; all other IPv4 sources are dropped.
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
			if err := must("-A", stagingChain, "-p", "tcp", "-s", c, "--dport", "9443", "-j", "ACCEPT"); err != nil {
				return fmt.Errorf("iptables fallback service uplink allow %s: %w", c, err)
			}
		}
		if err := denyTCP("9443"); err != nil {
			return fmt.Errorf("iptables fallback service uplink drop: %w", err)
		}
		if lp := ndconfig.LampacPort(); lp != "" && lp != "0" {
			// Loopback is accepted by the interface-scoped -i lo rule below;
			// never trust a 127/8 source address arriving on a WAN interface.
			if err := denyTCP(lp); err != nil {
				return fmt.Errorf("iptables fallback Lampac WAN drop: %w", err)
			}
		}
		mtls := ndconfig.AgentMTLSPort()
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
			if err := must("-A", stagingChain, "-p", "tcp", "-s", c, "--dport", mtls, "-j", "ACCEPT"); err != nil {
				return fmt.Errorf("iptables fallback mtls allow %s: %w", c, err)
			}
		}
		if err := must("-A", stagingChain, "-p", "tcp", "--dport", mtls, "-j", "DROP"); err != nil {
			return fmt.Errorf("iptables fallback mtls drop: %w", err)
		}
	} else {
		for _, port := range []string{"8787", "8788", "8789", "80", "8444", "9443", ndconfig.AgentMTLSPort(), ndconfig.RedirectHTTPSPort()} {
			if err := denyTCP(port); err != nil {
				return fmt.Errorf("iptables fallback deny %s: %w", port, err)
			}
		}
		if lp := ndconfig.LampacPort(); lp != "" && lp != "0" {
			if err := denyTCP(lp); err != nil {
				return fmt.Errorf("iptables fallback deny Lampac port %s: %w", lp, err)
			}
		}
		if err := must("-A", stagingChain, "-p", "udp", "--dport", ndconfig.RedirectHTTPSPort(), "-j", "DROP"); err != nil {
			return fmt.Errorf("iptables fallback deny primary redirect UDP %s: %w", ndconfig.RedirectHTTPSPort(), err)
		}
	}
	if err := must("-I", stagingChain, "1", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("iptables fallback established: %w", err)
	}
	if err := must("-I", stagingChain, "2", "-i", "lo", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("iptables fallback lo: %w", err)
	}

	// The chain is complete. Replace the existing managed jump in one kernel
	// operation; if none existed, install the first managed jump now.
	if jumpPos > 0 {
		if err := must("-R", "INPUT", strconv.Itoa(jumpPos), "-j", stagingChain); err != nil {
			return fmt.Errorf("iptables fallback: atomic INPUT jump replacement: %w", err)
		}
	} else {
		// Keep the managed chain behind both SSH safeguards during bootstrap.
		jumpAt := "2"
		if bootstrapSSH { jumpAt = "3" }
		if err := must("-I", "INPUT", jumpAt, "-j", stagingChain); err != nil {
			return fmt.Errorf("iptables fallback: install INPUT jump after SSH safeguard: %w", err)
		}
	}

	// Verify the live INPUT layout after the switch before removing the
	// temporary SSH safeguard. This catches stale positions, unexpected
	// concurrent rule changes, and a switch that did not become effective.
	verifiedInput, verifyErr := exec.Command("iptables", "-S", "INPUT").CombinedOutput()
	if verifyErr != nil {
		return fmt.Errorf("iptables fallback: verify switched INPUT: %v (%s)", verifyErr, truncate(string(verifiedInput), 200))
	}
	verifiedPos, verifiedChain, layoutErr := managedInputJump(string(verifiedInput))
	if layoutErr != nil {
		return fmt.Errorf("iptables fallback: verify switched INPUT layout: %w", layoutErr)
	}
	if verifiedPos == 0 || verifiedChain != stagingChain {
		return fmt.Errorf("iptables fallback: INPUT switch not verified (active=%q want=%q)", verifiedChain, stagingChain)
	}
	if !inputStartsWithSSHAllow(string(verifiedInput), sp) {
		return fmt.Errorf("iptables fallback: leading SSH safeguard is missing after INPUT switch")
	}
	if bootstrapSSH {
		if !iptablesHasExactRule(string(verifiedInput), "INPUT",
			[]string{"-p", "tcp", "--dport", "22", "-m", "comment", "--comment", "netductor-ssh-bootstrap", "-j", "ACCEPT"},
			[]string{"-p", "tcp", "-m", "tcp", "--dport", "22", "-m", "comment", "--comment", "netductor-ssh-bootstrap", "-j", "ACCEPT"},
		) {
			return fmt.Errorf("iptables fallback: temporary factory SSH safeguard missing after INPUT switch")
		}
	}
	verifiedRules, chainVerifyErr := exec.Command("iptables", "-S", stagingChain).CombinedOutput()
	if chainVerifyErr != nil {
		return fmt.Errorf("iptables fallback: verify active chain %s: %v (%s)", stagingChain, chainVerifyErr, truncate(string(verifiedRules), 200))
	}
	chainRules := string(verifiedRules)
	sshRuleOK := iptablesHasExactRule(chainRules, stagingChain,
		[]string{"-p", "tcp", "--dport", sp, "-j", "ACCEPT"},
		[]string{"-p", "tcp", "-m", "tcp", "--dport", sp, "-j", "ACCEPT"},
	)
	establishedRuleOK := iptablesHasExactRule(chainRules, stagingChain,
		[]string{"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
		[]string{"-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
	)
	if !sshRuleOK || !establishedRuleOK {
		return fmt.Errorf("iptables fallback: active chain %s failed exact essential-rule verification (ssh=%t established=%t)", stagingChain, sshRuleOK, establishedRuleOK)
	}

	// The leading SSH safeguard is intentionally retained across applies.
	fmt.Fprintf(os.Stderr, "firewall: iptables %s chain applied role=%s (fallback; prefer ufw)\n", stagingChain, role)
	if out, saveErr := exec.Command("netfilter-persistent", "save").CombinedOutput(); saveErr != nil {
		return fmt.Errorf("iptables fallback: persist firewall via netfilter-persistent: %v (%s)", saveErr, truncate(string(out), 200))
	}
	if err := persistRole(role); err != nil {
		return err
	}
	return nil
}

// Collect returns firewall status for doctor/API.
func Collect(role string) Status {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "core" {
		role = "primary"
	}
	if role == "" {
		role = detectRole()
	}
	if role != "primary" && role != "secondary" {
		return Status{
			Backend: BackendNone,
			Role: role,
			OK: false,
			Detail: "cannot determine a valid firewall role",
			Warnings: []string{"firewall health check refused an unknown role"},
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}
	st := Status{
		Role:      role,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		OpenWAN:   []string{sshPort() + "/tcp", "443/tcp"},
		DenyWAN:   []string{"22/tcp", "10050/tcp"},
	}
	if role == "primary" {
		st.OpenWAN = append(st.OpenWAN, ndconfig.RedirectHTTPSPort()+"/tcp", "80/tcp")
		st.Restricted = append(st.Restricted, ndconfig.AgentMTLSPort()+"/tcp ← "+ndconfig.SvcSPCIDR()+","+ndconfig.SvcPSCIDR()+",lo")
		st.DenyWAN = append(st.DenyWAN, "8788/tcp")
	} else {
		st.DenyWAN = append(st.DenyWAN, "8787/tcp", "8788/tcp", "8789/tcp", "80/tcp")
	}
	if wss, err := wssExpectations(role); err != nil {
		st.Warnings = append(st.Warnings, "WSS peer inventory: "+err.Error())
	} else {
		for _, peer := range wss {
			st.Restricted = append(st.Restricted, peer.Port+"/tcp ← "+peer.Source)
		}
	}

	if _, err := exec.LookPath("ufw"); err == nil {
		st.Backend = BackendUFW
		out, statusErr := exec.Command("ufw", "status", "numbered").CombinedOutput()
		s := string(out)
		if statusErr != nil {
			st.Active = false
			st.Detail = firstLines(s, 16)
			st.Warnings = append(st.Warnings, "cannot read UFW numbered status: "+truncate(statusErr.Error(), 160)+" ("+truncate(s, 160)+")")
			st.OK = false
			return st
		}
		st.Active = strings.Contains(s, "Status: active")
		st.Detail = firstLines(s, 16)
		ok, warns := ufwEffectivePolicy(s, role)
		if _, ipv6Err := ufwIPv6Enabled(); ipv6Err != nil {
			ok = false
			warns = append(warns, "IPv6 filtering prerequisite: "+ipv6Err.Error())
		}
		verbose, verboseErr := exec.Command("ufw", "status", "verbose").CombinedOutput()
		if verboseErr != nil {
			ok = false
			warns = append(warns, "cannot read UFW verbose status: "+truncate(string(verbose), 160))
		} else if defaultOK, reason := ufwDefaultIncomingDenied(string(verbose)); !defaultOK {
			ok = false
			warns = append(warns, reason)
		}
		st.OK = st.Active && ok
		st.Warnings = append(st.Warnings, warns...)
		if !st.Active {
			st.Warnings = append(st.Warnings, "ufw installed but inactive")
			st.OK = false
		}
		return st
	}
	if _, err := exec.LookPath("iptables"); err == nil {
		st.Backend = BackendIPTables
		input, inputErr := exec.Command("iptables", "-S", "INPUT").CombinedOutput()
		jumpPos, activeChain, jumpErr := managedInputJump(string(input))
		_ = jumpPos
		if inputErr == nil && jumpErr == nil && activeChain != "" {
			out, chainErr := exec.Command("iptables", "-S", activeChain).CombinedOutput()
			s := string(out)
			st.Active = chainErr == nil && strings.Contains(s, activeChain)
			st.Detail = firstLines(s, 20)
			if !st.Active || !strings.Contains(s, "--dport "+sshPort()) {
				st.Warnings = append(st.Warnings, "iptables managed chain missing or incomplete (ufw preferred)")
			} else {
				st.Warnings = append(st.Warnings, "iptables fallback: policy not fully verified — not OK (prefer ufw)")
			}
		} else {
			st.Active = false
			st.Detail = firstLines(string(input), 20)
			st.Warnings = append(st.Warnings, "iptables managed INPUT jump missing or ambiguous")
		}
		// Do not report OK for iptables fallback — role policy is not fully verified.
		st.OK = false
		return st
	}
	st.Backend = BackendNone
	st.OK = false
	st.Detail = "no ufw and no iptables"
	st.Warnings = append(st.Warnings, "CRITICAL: no host firewall backend")
	return st
}


// ufwDefaultIncomingDenied requires an explicit restrictive default incoming policy.
// The setting is available in `ufw status verbose`, not reliably in numbered status.
func ufwDefaultIncomingDenied(status string) (bool, string) {
	var defaultLine string
	count := 0
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "default:") {
			count++
			defaultLine = line
		}
	}
	if count != 1 {
		return false, fmt.Sprintf("UFW verbose status must contain exactly one Default policy line (found %d)", count)
	}
	colon := strings.IndexByte(defaultLine, ':')
	if colon < 0 {
		return false, "malformed UFW default policy line"
	}
	policies := make(map[string]string)
	for _, item := range strings.Split(strings.TrimSpace(defaultLine[colon+1:]), ",") {
		item = strings.TrimSpace(item)
		open := strings.Index(item, "(")
		close := strings.Index(item, ")")
		if open <= 0 || close <= open {
			continue
		}
		direction := strings.ToLower(strings.TrimSpace(item[open+1 : close]))
		policy := strings.ToLower(strings.TrimSpace(item[:open]))
		if _, exists := policies[direction]; exists {
			return false, "duplicate UFW default policy for " + direction
		}
		policies[direction] = policy
	}
	expected := []struct{ direction, policy string }{
		{"incoming", "deny"},
		{"outgoing", "allow"},
		{"routed", "deny"},
	}
	for _, item := range expected {
		got, found := policies[item.direction]
		if !found {
			return false, "default " + item.direction + " policy is missing from UFW verbose status"
		}
		if got != item.policy && !(item.direction == "incoming" && got == "reject") {
			return false, "default " + item.direction + " policy is " + got + " (want " + item.policy + ")"
		}
	}
	return true, ""
}

// ufwRule is one line from `ufw status numbered`.
type ufwRule struct {
	Num    int
	Port   string
	Proto  string
	Action    string
	Direction string
	From      string
	IPv6      bool
	Raw    string
}

// parseUfwNumbered preserves address-family identity. UFW marks IPv6 rules with
// "(v6)" after the destination and source; removing that marker before parsing
// keeps the source usable while IPv6 remains explicit in the rule.
func parseUfwNumbered(status string) []ufwRule {
	var rules []ufwRule
	for _, ln := range strings.Split(status, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || !strings.HasPrefix(ln, "[") {
			continue
		}
		rb := strings.Index(ln, "]")
		if rb < 0 {
			continue
		}
		numStr := strings.TrimSpace(ln[1:rb])
		num, err := strconv.Atoi(numStr)
		if err != nil || num < 1 {
			continue
		}
		rest := strings.TrimSpace(ln[rb+1:])
		ipv6 := strings.Contains(rest, "(v6)")
		rest = strings.ReplaceAll(rest, " (v6)", "")
		if i := strings.Index(rest, "#"); i >= 0 {
			rest = strings.TrimSpace(rest[:i])
		}
		fields := strings.Fields(rest)
		if len(fields) < 3 {
			continue
		}
		portProto := fields[0]
		port, proto := portProto, ""
		if j := strings.Index(portProto, "/"); j >= 0 {
			port, proto = portProto[:j], portProto[j+1:]
		}
		actionIdx := -1
		for i, field := range fields {
			switch field {
			case "ALLOW", "DENY", "REJECT", "LIMIT":
				actionIdx = i
			}
			if actionIdx >= 0 {
				break
			}
		}
		if actionIdx < 0 {
			continue
		}
		// Direction is security-significant. Do not silently treat an
		// unrecognized/missing direction as inbound; the caller's completeness
		// check will then reject the numbered listing.
		if actionIdx+1 >= len(fields) {
			continue
		}
		direction := fields[actionIdx+1]
		if direction != "IN" && direction != "OUT" && direction != "FWD" {
			continue
		}
		fromParts := fields[actionIdx+2:]
		from := strings.Join(fromParts, " ")
		if from == "" {
			from = "Anywhere"
		}
		rules = append(rules, ufwRule{
			Num: num, Port: port, Proto: proto, Action: fields[actionIdx],
			Direction: direction, From: from, IPv6: ipv6, Raw: ln,
		})
	}
	return rules
}

func ufwRuleAllowsTraffic(action string) bool {
	// UFW LIMIT is rate-limited acceptance, not a deny. Treat it as an
	// allow when checking that restricted ports have no exposure exceptions.
	return action == "ALLOW" || action == "LIMIT"
}

func ufwRuleMatchesPortProtocol(r ufwRule, port, protocol string) bool {
	// UFW status also lists OUT and FWD rules; they must never influence
	// inbound policy decisions.
	if r.Direction != "IN" {
		return false
	}
	// An empty requested protocol matches any protocol; an omitted UFW
	// protocol matches TCP as well as other protocols.
	gotProtocol := strings.ToLower(strings.TrimSpace(r.Proto))
	wantProtocol := strings.ToLower(strings.TrimSpace(protocol))
	if wantProtocol != "" && gotProtocol != "" && gotProtocol != wantProtocol {
		return false
	}
	rp := strings.TrimSpace(r.Port)
	if rp == "" || rp == "Anywhere" || rp == "Anywhere (v6)" || rp == "*" || strings.EqualFold(rp, "any") {
		return true
	}
	for _, sep := range []string{":", "-"} {
		if i := strings.Index(rp, sep); i > 0 {
			lo, hi := rp[:i], rp[i+len(sep):]
			a, ea := strconv.Atoi(lo)
			b, eb := strconv.Atoi(hi)
			pn, ep := strconv.Atoi(port)
			if ea == nil && eb == nil && ep == nil && pn >= a && pn <= b {
				return true
			}
		}
	}
	if j := strings.Index(rp, "/"); j >= 0 {
		rp = rp[:j]
	}
	return rp == port
}

func ufwRuleMatchesPortTCP(r ufwRule, port string) bool {
	return ufwRuleMatchesPortProtocol(r, port, "tcp")
}

func ufwFromIsWAN(from string) bool {
	from = strings.TrimSpace(from)
	return from == "" || from == "Anywhere" || from == "Anywhere (v6)" || strings.EqualFold(from, "any")
}

func cidrEqual(a, b string) bool {
	_, na, ea := net.ParseCIDR(strings.TrimSpace(a))
	_, nb, eb := net.ParseCIDR(strings.TrimSpace(b))
	if ea != nil || eb != nil {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	oa, _ := na.Mask.Size()
	ob, _ := nb.Mask.Size()
	return oa == ob && na.IP.Equal(nb.IP)
}

// firstTCPActionFromWANFamily returns the first matching WAN action for one
// address family. Rule order is evaluated independently for IPv4 and IPv6.
func firstTCPActionFromWANFamily(rules []ufwRule, port string, ipv6 bool) (action string, found bool) {
	for _, rule := range rules {
		if rule.IPv6 != ipv6 || !ufwRuleMatchesPortTCP(rule, port) || !ufwFromIsWAN(rule.From) {
			continue
		}
		return rule.Action, true
	}
	return "", false
}

// ufwAllowsBeforeFirstWANAction finds source-specific/global ALLOW rules that
// precede the first WAN rule for each address family. Later ALLOW rules cannot
// override an earlier matching DENY and are retained as administrator state.
func ufwAllowsBeforeFirstWANAction(rules []ufwRule, port, wanAction string) []ufwRule {
	var conflicts []ufwRule
	for _, ipv6 := range []bool{false, true} {
		boundary := int(^uint(0) >> 1)
		for _, rule := range rules {
			if rule.IPv6 == ipv6 && ufwRuleMatchesPortTCP(rule, port) && ufwFromIsWAN(rule.From) {
				if rule.Action == wanAction { boundary = rule.Num }
				break
			}
		}
		for _, rule := range rules {
			if rule.IPv6 == ipv6 && rule.Num < boundary &&
				ufwRuleMatchesPortTCP(rule, port) && ufwRuleAllowsTraffic(rule.Action) {
				conflicts = append(conflicts, rule)
			}
		}
	}
	return conflicts
}

// firstTCPActionFromWAN preserves the legacy IPv4 helper contract.
func firstTCPActionFromWAN(rules []ufwRule, port string) (action string, found bool) {
	return firstTCPActionFromWANFamily(rules, port, false)
}

// ufwSourceCovers reports whether a UFW source selector covers every address
// in targetCIDR. "Anywhere" covers all sources; host addresses are treated as
// single-address networks.
func ufwSourceCovers(from, targetCIDR string) bool {
	from = strings.TrimSpace(from)
	if ufwFromIsWAN(from) {
		return true
	}
	targetIP, targetNet, targetErr := net.ParseCIDR(strings.TrimSpace(targetCIDR))
	if targetErr != nil {
		if ip := net.ParseIP(strings.TrimSpace(targetCIDR)); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			targetNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
		} else {
			return false
		}
	} else if targetIP == nil {
		return false
	}
	fromIP, fromNet, fromErr := net.ParseCIDR(from)
	if fromErr != nil {
		if ip := net.ParseIP(from); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			fromNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
		} else {
			return false
		}
	} else if fromIP == nil {
		return false
	}
	if (targetNet.IP.To4() != nil) != (fromNet.IP.To4() != nil) {
		return false
	}
	ones, _ := fromNet.Mask.Size()
	targetOnes, _ := targetNet.Mask.Size()
	return ones <= targetOnes && fromNet.Contains(targetNet.IP)
}

// ufwSourceOverlaps reports whether a UFW source selector intersects the
// requested source range. It is deliberately conservative for partial overlap.
// ufwSourceContainedBy reports whether every source address selected by
// from is contained in allowedCIDR.
func ufwSourceContainedBy(from, allowedCIDR string) bool {
	return ufwSourceCovers(allowedCIDR, from)
}

func ufwSourceOverlaps(from, targetCIDR string) bool {
	from = strings.TrimSpace(from)
	if ufwFromIsWAN(from) {
		return true
	}
	parseNet := func(value string) (*net.IPNet, bool) {
		ip, network, err := net.ParseCIDR(value)
		if err == nil && ip != nil {
			return network, true
		}
		ip = net.ParseIP(value)
		if ip == nil {
			return nil, false
		}
		bits := 128
		if ip.To4() != nil {
			bits = 32
		}
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, true
	}
	a, okA := parseNet(from)
	b, okB := parseNet(strings.TrimSpace(targetCIDR))
	if !okA || !okB || (a.IP.To4() != nil) != (b.IP.To4() != nil) {
		return false
	}
	return a.Contains(b.IP) || b.Contains(a.IP)
}

// firstTCPActionFromSource returns the first ordered rule that affects the
// requested source range. A partial overlap is returned as a non-ALLOW action
// so a narrow earlier rule cannot be silently ignored.
func firstTCPActionFromSource(rules []ufwRule, port, sourceCIDR string, ipv6 bool) (action string, found bool) {
	for _, rule := range rules {
		if rule.IPv6 != ipv6 || !ufwRuleMatchesPortTCP(rule, port) {
			continue
		}
		if ufwSourceCovers(rule.From, sourceCIDR) {
			return rule.Action, true
		}
		if ufwSourceOverlaps(rule.From, sourceCIDR) {
			return "PARTIAL_" + rule.Action, true
		}
	}
	return "", false
}

func ufwEffectivePolicy(status, role string) (ok bool, warnings []string) {
	ok = true
	rules := parseUfwNumbered(status)
	numberedLines := 0
	for _, line := range strings.Split(status, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			numberedLines++
		}
	}
	if len(rules) == 0 || len(rules) != numberedLines {
		return false, []string{fmt.Sprintf("cannot completely parse numbered UFW rules: parsed %d of %d; address-family policy is unverifiable", len(rules), numberedLines)}
	}
	requireWAN := func(port, wantAction, label string, ipv6 bool) {
		act, found := firstTCPActionFromWANFamily(rules, port, ipv6)
		family := "IPv4"
		if ipv6 {
			family = "IPv6"
		}
		if !found {
			ok = false
			warnings = append(warnings, label+": no "+port+"/tcp WAN rule for "+family)
			return
		}
		if act != wantAction {
			ok = false
			warnings = append(warnings, label+": first "+family+" WAN rule for "+port+"/tcp is "+act+" (want "+wantAction+")")
		}
	}
	requireBoth := func(port, action, label string) {
		requireWAN(port, action, label, false)
		requireWAN(port, action, label, true)
	}
	rejectAnyAllow := func(port, label string) {
		for _, rule := range rules {
			if ufwRuleMatchesPortTCP(rule, port) && ufwRuleAllowsTraffic(rule.Action) {
				ok = false
				family := "IPv4"
				if rule.IPv6 { family = "IPv6" }
				warnings = append(warnings, label+": unexpected "+family+" ALLOW from "+rule.From+" for "+port+"/tcp")
			}
		}
	}
	rejectAnyDeny := func(port, label string) {
		for _, rule := range rules {
			if ufwRuleMatchesPortTCP(rule, port) && (rule.Action == "DENY" || rule.Action == "REJECT") {
				ok = false
				warnings = append(warnings, label+": unexpected "+rule.Action+" from "+rule.From+" for "+port+"/tcp")
			}
		}
	}
	sp := sshPort()
	for _, item := range []struct{ port, action, label string }{
		{sp, "ALLOW", "ssh"},
		{"443", "ALLOW", "vless"},
		{"22", "DENY", "ssh22"},
		{"10050", "DENY", "zabbix"},
		{"10051", "DENY", "zabbix"},
	} {
		requireBoth(item.port, item.action, item.label)
	}
	for _, item := range []struct{ port, label string }{
		{"10050", "zabbix"}, {"10051", "zabbix"},
	} {
		rejectAnyAllow(item.port, item.label)
	}
	for _, rule := range ufwAllowsBeforeFirstWANAction(rules, "22", "DENY") {
		ok = false
		warnings = append(warnings, "ssh22: effective "+rule.Action+" from "+rule.From+" precedes the WAN DENY")
	}
	rejectAnyDeny("443", "vless")
	// Do not let a source-specific deny be hidden by a later global SSH
	// allow: it may still match the active administrator connection.
	for _, rule := range rules {
		if ufwRuleMatchesPortTCP(rule, sp) && (rule.Action == "DENY" || rule.Action == "REJECT") {
			ok = false
			warnings = append(warnings, "ssh: unexpected "+rule.Action+" from "+rule.From+" for "+sp+"/tcp")
		}
	}

	if role == "primary" {
		for _, item := range []struct{ port, action, label string }{
			{ndconfig.RedirectHTTPSPort(), "ALLOW", "redirect"},
			{"80", "ALLOW", "http-acme"},
			{"8787", "DENY", "secondary-api"},
			{"8445", "DENY", "secondary-wss"},
			{"8788", "DENY", "plain-agent"},
			{"9443", "DENY", "service-uplink-wan"},
		} {
			requireBoth(item.port, item.action, item.label)
		}
		for _, item := range []struct{ port, label string }{
			{ndconfig.RedirectHTTPSPort(), "redirect"}, {"80", "http-acme"},
		} { rejectAnyDeny(item.port, item.label) }
		for _, item := range []struct{ port, label string }{
			{"8787", "secondary-api"}, {"8445", "secondary-wss"}, {"8788", "plain-agent"},
		} {
			rejectAnyAllow(item.port, item.label)
		}

		// The configured service-path CIDRs are IPv4. They must not be
		// interpreted as permission to expose the same port over IPv6.
		allowedUplinkSources := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()}
		for _, cidr := range allowedUplinkSources {
			action, found := firstTCPActionFromSource(rules, "9443", cidr, false)
			if !found || action != "ALLOW" {
				ok = false
				warnings = append(warnings, "service uplink: first matching IPv4 rule for "+cidr+" on 9443/tcp is "+action+" (want ALLOW)")
			}
		}
		for _, rule := range rules {
			if !ufwRuleMatchesPortTCP(rule, "9443") || !ufwRuleAllowsTraffic(rule.Action) {
				continue
			}
			if rule.IPv6 {
				ok = false
				warnings = append(warnings, "service uplink: unexpected IPv6 ALLOW from "+rule.From+" for 9443/tcp")
				continue
			}
			trusted := false
			for _, source := range allowedUplinkSources {
				if ufwSourceContainedBy(rule.From, source) {
					trusted = true
					break
				}
			}
			if !trusted {
				ok = false
				warnings = append(warnings, "service uplink: IPv4 ALLOW from untrusted source "+rule.From+" for 9443/tcp")
			}
		}

		mtls := ndconfig.AgentMTLSPort()
		for _, source := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
			action, found := firstTCPActionFromSource(rules, mtls, source, false)
			if !found || action != "ALLOW" {
				ok = false
				warnings = append(warnings, "mtls: first matching IPv4 rule for "+source+" on "+mtls+"/tcp is "+action+" (want ALLOW)")
			}
		}
		// No IPv6 source is trusted for the current mTLS service topology.
		// Reject every IPv6 ALLOW, including narrow source-specific exceptions.
		for _, rule := range rules {
			if rule.IPv6 && ufwRuleMatchesPortTCP(rule, mtls) && ufwRuleAllowsTraffic(rule.Action) {
				ok = false
				warnings = append(warnings, "mtls: unexpected IPv6 ALLOW from "+rule.From+" for "+mtls+"/tcp")
			}
		}
		// IPv4 mTLS must be reachable only from the configured service CIDRs.
		// Loopback is handled by UFW's interface-scoped built-in lo rules.
		allowedMTLSSources := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()}
		for _, rule := range rules {
			if rule.IPv6 || !ufwRuleMatchesPortTCP(rule, mtls) || !ufwRuleAllowsTraffic(rule.Action) {
				continue
			}
			trusted := false
			for _, source := range allowedMTLSSources {
				if ufwSourceContainedBy(rule.From, source) {
					trusted = true
					break
				}
			}
			if !trusted {
				ok = false
				warnings = append(warnings, "mtls: IPv4 ALLOW from untrusted source "+rule.From+" for "+mtls+"/tcp")
			}
		}

		lp := ndconfig.LampacPort()
		if lp != "" && lp != "0" {
			requireBoth(lp, "DENY", "lampac-wan")
			for _, rule := range rules {
				if ufwRuleMatchesPortTCP(rule, lp) && ufwRuleAllowsTraffic(rule.Action) {
					ok = false
					warnings = append(warnings, "lampac: unexpected source-based ALLOW from "+rule.From+" for "+lp+"/tcp")
				}
			}
		}
	} else {
		requireBoth("80", "DENY", "secondary-http")
		for _, port := range []string{"8787", "8788", "8789"} {
			requireBoth(port, "DENY", "secondary-"+port)
		}
		for _, port := range []string{"80", "8787", "8788", "8789"} {
			rejectAnyAllow(port, "secondary-restricted")
		}
		primaryOnly := []struct{ port, label string }{
			{"8444", "primary-wss"}, {"9443", "primary-service-uplink"},
			{ndconfig.AgentMTLSPort(), "primary-mtls"},
			{ndconfig.RedirectHTTPSPort(), "primary-redirect"},
		}
		if lp := ndconfig.LampacPort(); lp != "" && lp != "0" {
			primaryOnly = append(primaryOnly, struct{ port, label string }{lp, "primary-lampac"})
		}
		for _, item := range primaryOnly {
			for _, rule := range rules {
				if ufwRuleMatchesPortProtocol(rule, item.port, "") && ufwRuleAllowsTraffic(rule.Action) {
					ok = false
					warnings = append(warnings, item.label+": unexpected "+rule.Proto+" ALLOW from "+rule.From+" for "+item.port)
				}
			}
		}
	}
	if err := verifyWSSRules(rules, role); err != nil {
		ok = false
		warnings = append(warnings, "WSS peer policy: "+err.Error())
	}
	return ok, warnings
}

// legacy helpers kept for tests
func ufwLineMentionsPortProto(ln, port, proto string) bool {
	port = strings.TrimSpace(port)
	proto = strings.TrimSpace(proto)
	if port == "" {
		return false
	}
	fields := strings.Fields(ln)
	for _, f := range fields {
		f = strings.TrimSuffix(f, "(v6)")
		if proto == "tcp" {
			if f == port+"/tcp" || f == port {
				return true
			}
		}
		if f == port+"/"+proto {
			return true
		}
	}
	return false
}

func ufwLineMentionsPort(ln, port string) bool {
	return ufwLineMentionsPortProto(ln, port, "tcp")
}

func ufwLineAnyPort(lines []string, port string) bool {
	for _, ln := range lines {
		if ufwLineMentionsPort(ln, port) {
			return true
		}
	}
	return false
}

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// HealIfNeeded re-applies rules when Collect reports !OK.
// Safe-ish: only reinforces known allow/deny list; does not open new WAN ports.
// Disabled unless NETDUCTOR_FW_AUTOHEAL=1 (or force=true from explicit CLI/UI).
func HealIfNeeded(role string, force bool) (healed bool, err error) {
	if !force && os.Getenv("NETDUCTOR_FW_AUTOHEAL") != "1" {
		return false, nil
	}
	st := Collect(role)
	if st.OK {
		return false, nil
	}
	if err := ApplyRole(role); err != nil {
		return false, err
	}
	return true, nil
}

// AlertFunc optional hook (wired from main/metrics to notify.AlertOnce) to avoid import cycles.
var AlertFunc func(key, msg string)

// AlertIfUnhealthy reports Collect() and alerts when !OK (cooldown via AlertFunc).
func AlertIfUnhealthy() Status {
	st := Collect("")
	if st.OK || AlertFunc == nil {
		return st
	}
	msg := "🔥 Firewall not OK: backend=" + string(st.Backend) + " active="
	if st.Active {
		msg += "true"
	} else {
		msg += "false"
	}
	msg += " role=" + st.Role
	if len(st.Warnings) > 0 {
		msg += " — " + st.Warnings[0]
	}
	AlertFunc("firewall:not-ok", msg)
	return st
}
