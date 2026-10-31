// Package firewall applies role-based host firewall for netductor nodes.
// Preferred backend: ufw (installed if missing). Fallback: iptables-nft or legacy iptables.
// Never silently skip: Apply returns error if no backend can enforce rules.
package firewall

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/PavelNeyman/netductor/internal/format"

	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
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
	out, err := cmd.CombinedOutput()
	if err == nil {
		if _, err2 := exec.LookPath("ufw"); err2 == nil {
			return BackendUFW, nil
		}
	}
	// fallback: iptables
	if _, err := exec.LookPath("iptables"); err == nil {
		fmt.Fprintf(os.Stderr, "firewall: ufw install failed (%v: %s) — using iptables fallback\n", err, truncate(string(out), 200))
		return BackendIPTables, nil
	}
	return BackendNone, fmt.Errorf("firewall: cannot install ufw and no iptables: %v %s", err, truncate(string(out), 300))
}

func truncate(s string, n int) string { return format.Truncate(strings.TrimSpace(s), n) }


// acquireFirewallApplyLock serializes Netductor firewall mutations across processes.
// It does not coordinate with UFW, iptables, or other external firewall managers.
func acquireFirewallApplyLock() (func(), error) {
	const lockPath = "/run/lock/netductor-firewall.lock"
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
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("firewall: acquire apply lock: %w", err)
	}
	return func() {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
	}, nil
}

// ApplyRole installs backend if needed and applies primary or secondary profile.
func ApplyRole(role string) error {
	unlock, err := acquireFirewallApplyLock()
	if err != nil {
		return err
	}
	defer unlock()

	role = strings.ToLower(strings.TrimSpace(role))
	if role == "core" {
		role = "primary"
	}
	if role != "primary" && role != "secondary" {
		role = detectRole()
	}
	be, err := EnsureInstalled()
	if err != nil {
		return err
	}
	switch be {
	case BackendUFW:
		return applyUFW(role)
	case BackendIPTables:
		return applyIPTables(role)
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

// DetectRoleStrict identifies a role only when a durable or live role marker exists.
// It returns ok=false rather than guessing primary when the host has no usable markers.
func DetectRoleStrict() (string, bool) {
	// Live secondary agent is definitive — prevents wrong role=primary after node upgrade paths.
	if st, _ := exec.Command("systemctl", "is-active", "netductor-secondary-agent").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "secondary", true
	}
	// Explicit role file (written by install / firewall apply).
	if b, err := os.ReadFile("/etc/netductor/role"); err == nil {
		s := strings.TrimSpace(strings.ToLower(string(b)))
		if s == "secondary" {
			return "secondary", true
		}
		if s == "primary" || s == "core" {
			return "primary", true
		}
	}
	// Primary control-plane markers (bundle.json may exist on primary after export — do NOT treat as secondary).
	if _, err := os.Stat("/var/lib/netductor/READY.txt"); err == nil {
		return "primary", true
	}
	if st, _ := exec.Command("systemctl", "is-active", "netductor-api").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "primary", true
	}
	if st, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "primary", true
	}
	// Last resort: secondary bundle only if no primary markers above.
	if _, err := os.Stat("/var/lib/netductor/secondary/bundle.json"); err == nil {
		return "secondary", true
	}
	return "", false
}

// persistRole writes /etc/netductor/role so status matches last apply.
func persistRole(role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "primary" && role != "secondary" {
		return fmt.Errorf("firewall: refusing to persist invalid role %q", role)
	}
	const dir = "/etc/netductor"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("firewall: create role directory: %w", err)
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
	checkSource := func(port, source, label string) {
		action, found := firstTCPActionFromSource(rules, port, source, false)
		if found && action != "ALLOW" {
			conflicts = append(conflicts, fmt.Sprintf("%s %s first rule for %s is %s, want ALLOW", label, port, source, action))
		}
	}
	for _, item := range []struct{ port, action, label string }{
		{"443", "ALLOW", "vless"}, {"22", "DENY", "ssh22"},
		{"10050", "DENY", "zabbix"}, {"10051", "DENY", "zabbix"},
	} {
		checkWAN(item.port, item.action, item.label)
	}
	if role == "primary" {
		for _, item := range []struct{ port, action, label string }{
			{ndconfig.RedirectHTTPSPort(), "ALLOW", "redirect"},
			{"80", "ALLOW", "http-acme"}, {"8444", "ALLOW", "primary-wss"},
			{"8788", "DENY", "plain-agent"}, {"9443", "DENY", "service-uplink"},
		} { checkWAN(item.port, item.action, item.label) }
		uplink := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()}
		for _, source := range uplink { checkSource("9443", source, "service-uplink") }
		mtls := ndconfig.AgentMTLSPort()
		trustedMTLS := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"}
		for _, source := range trustedMTLS { checkSource(mtls, source, "mtls") }
		for _, rule := range rules {
			if !ufwRuleMatchesPortTCP(rule, "9443") || rule.Action != "ALLOW" { continue }
			trusted := false
			if !rule.IPv6 {
				for _, source := range uplink {
					if ufwSourceContainedBy(rule.From, source) { trusted = true; break }
				}
			}
			if !trusted { conflicts = append(conflicts, "service-uplink has an existing untrusted ALLOW from "+rule.From) }
		}
		for _, rule := range rules {
			if !ufwRuleMatchesPortTCP(rule, mtls) || rule.Action != "ALLOW" { continue }
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
			checkSource(lp, "127.0.0.1", "lampac-loopback")
			for _, rule := range rules {
				if ufwRuleMatchesPortTCP(rule, lp) && rule.Action == "ALLOW" &&
					(rule.IPv6 || !ufwSourceContainedBy(rule.From, "127.0.0.1")) {
					conflicts = append(conflicts, "lampac has an existing untrusted ALLOW from "+rule.From)
				}
			}
		}
	} else {
		checkWAN("8445", "ALLOW", "secondary-wss")
		for _, port := range []string{"80", "8787", "8788", "8789"} {
			checkWAN(port, "DENY", "secondary-restricted")
		}
	}
	if len(conflicts) != 0 {
		return fmt.Errorf("existing UFW policy conflicts; refusing mutation: %s", strings.Join(conflicts, "; "))
	}
	return nil
}

func applyUFW(role string) error {
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
	if err := ufwPreflightPolicy(string(current), role); err != nil {
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
		if err := must("allow", "8444/tcp", "comment", "nd-wss-sp"); err != nil { return fmt.Errorf("allow primary WSS port: %w", err) }
		if err := must("deny", "8788/tcp", "comment", "no-plain-agent"); err != nil {
			return err
		}
		mtls := ndconfig.AgentMTLSPort()
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"} {
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
			if err := must("allow", "from", "127.0.0.1", "to", "any", "port", lp, "proto", "tcp", "comment", "lampac-local"); err != nil { return fmt.Errorf("allow Lampac loopback: %w", err) }
		}
	} else {
		if err := must("allow", "8445/tcp", "comment", "nd-wss-ps"); err != nil { return fmt.Errorf("allow secondary WSS port: %w", err) }
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
	if ok, warnings := ufwEffectivePolicy(string(status), role); !ok {
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
	for _, line := range strings.Split(rules, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "-A" || fields[1] != "INPUT" {
			continue
		}
		position++
		for i := 2; i+1 < len(fields); i++ {
			if fields[i] != "-j" && fields[i] != "--jump" &&
				fields[i] != "-g" && fields[i] != "--goto" {
				continue
			}
			target := fields[i+1]
			if target != "NETDUCTOR" && target != "NETDUCTOR_A" && target != "NETDUCTOR_B" {
				continue
			}
			if chain != "" {
				return 0, "", fmt.Errorf("multiple Netductor INPUT jumps (%s and %s)", chain, target)
			}
			chain = target
			// Continue scanning to detect duplicate managed jumps.
			break
		}
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

func applyIPTables(role string) error {
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

	// A temporary direct SSH allow is installed only when an identical rule
	// is not already present. It remains in place if staging or switching fails.
	temporarySSHAllow := false
	if exec.Command("iptables", "-C", "INPUT", "-p", "tcp", "--dport", sp, "-j", "ACCEPT").Run() != nil {
		if err := must("-I", "INPUT", "1", "-p", "tcp", "--dport", sp, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("iptables fallback: temporary SSH safeguard: %w", err)
		}
		temporarySSHAllow = true
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
	if role == "primary" {
		redir := ndconfig.RedirectHTTPSPort()
		for _, port := range []string{redir, "80", "8444"} {
			if err := allowTCP(port); err != nil {
				return fmt.Errorf("iptables fallback allow %s: %w", port, err)
			}
		}
		if err := denyTCP("8788"); err != nil {
			return fmt.Errorf("iptables fallback deny 8788: %w", err)
		}
		mtls := ndconfig.AgentMTLSPort()
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"} {
			if err := must("-A", stagingChain, "-p", "tcp", "-s", c, "--dport", mtls, "-j", "ACCEPT"); err != nil {
				return fmt.Errorf("iptables fallback mtls allow %s: %w", c, err)
			}
		}
		if err := must("-A", stagingChain, "-p", "tcp", "--dport", mtls, "-j", "DROP"); err != nil {
			return fmt.Errorf("iptables fallback mtls drop: %w", err)
		}
	} else {
		if err := allowTCP("8445"); err != nil {
			return err
		}
		for _, port := range []string{"8787", "8788", "8789", "80"} {
			if err := denyTCP(port); err != nil {
				return fmt.Errorf("iptables fallback deny %s: %w", port, err)
			}
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
		if err := must("-I", "INPUT", "1", "-j", stagingChain); err != nil {
			return fmt.Errorf("iptables fallback: install INPUT jump: %w", err)
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
	verifiedRules, chainVerifyErr := exec.Command("iptables", "-S", stagingChain).CombinedOutput()
	if chainVerifyErr != nil {
		return fmt.Errorf("iptables fallback: verify active chain %s: %v (%s)", stagingChain, chainVerifyErr, truncate(string(verifiedRules), 200))
	}
	if !strings.Contains(string(verifiedRules), "-j ACCEPT") ||
		!strings.Contains(string(verifiedRules), "--dport "+sp) ||
		!strings.Contains(string(verifiedRules), "--ctstate ESTABLISHED,RELATED") {
		return fmt.Errorf("iptables fallback: active chain %s failed essential-rule verification", stagingChain)
	}

	// The new policy is now active and verified. Remove only the temporary SSH
	// exception created by this invocation; pre-existing administrator rules are kept.
	if temporarySSHAllow {
		if err := must("-D", "INPUT", "-p", "tcp", "--dport", sp, "-j", "ACCEPT"); err != nil {
			return fmt.Errorf("iptables fallback: remove temporary SSH safeguard: %w", err)
		}
	}
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
	if role == "" {
		role = detectRole()
	}
	st := Status{
		Role:      role,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		OpenWAN:   []string{sshPort() + "/tcp", "443/tcp"},
		DenyWAN:   []string{"22/tcp", "10050/tcp"},
	}
	if role == "primary" {
		st.OpenWAN = append(st.OpenWAN, ndconfig.RedirectHTTPSPort()+"/tcp", "8444/tcp", "80/tcp")
		st.Restricted = append(st.Restricted, ndconfig.AgentMTLSPort()+"/tcp ← "+ndconfig.SvcSPCIDR()+","+ndconfig.SvcPSCIDR()+",lo")
		st.DenyWAN = append(st.DenyWAN, "8788/tcp")
	} else {
		st.OpenWAN = append(st.OpenWAN, "8445/tcp")
		st.DenyWAN = append(st.DenyWAN, "8787/tcp", "8788/tcp", "8789/tcp", "80/tcp")
	}

	if _, err := exec.LookPath("ufw"); err == nil {
		st.Backend = BackendUFW
		out, _ := exec.Command("ufw", "status", "numbered").CombinedOutput()
		s := string(out)
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
	Action string
	From   string
	IPv6   bool
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
		fromParts := fields[actionIdx+1:]
		for len(fromParts) > 0 && (fromParts[0] == "IN" || fromParts[0] == "OUT") {
			fromParts = fromParts[1:]
		}
		from := strings.Join(fromParts, " ")
		if from == "" {
			from = "Anywhere"
		}
		rules = append(rules, ufwRule{
			Num: num, Port: port, Proto: proto, Action: fields[actionIdx],
			From: from, IPv6: ipv6, Raw: ln,
		})
	}
	return rules
}

func ufwRuleMatchesPortTCP(r ufwRule, port string) bool {
	if r.Proto == "udp" {
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

func ufwFromIsWAN(from string) bool {
	from = strings.TrimSpace(from)
	return from == "" || from == "Anywhere" || from == "Anywhere (v6)" || strings.EqualFold(from, "any")
}

func ufwFromIsTrustedMTLS(from string) bool {
	from = strings.TrimSpace(from)
	if from == "127.0.0.1" || from == "lo" || strings.HasPrefix(from, "127.0.0.1/") {
		return true
	}
	return cidrEqual(from, ndconfig.SvcSPCIDR()) || cidrEqual(from, ndconfig.SvcPSCIDR())
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

	if role == "primary" {
		for _, item := range []struct{ port, action, label string }{
			{ndconfig.RedirectHTTPSPort(), "ALLOW", "redirect"},
			{"80", "ALLOW", "http-acme"},
			{"8444", "ALLOW", "primary-wss"},
			{"8788", "DENY", "plain-agent"},
			{"9443", "DENY", "service-uplink-wan"},
		} {
			requireBoth(item.port, item.action, item.label)
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
			if !ufwRuleMatchesPortTCP(rule, "9443") || rule.Action != "ALLOW" {
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
		for _, source := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"} {
			action, found := firstTCPActionFromSource(rules, mtls, source, false)
			if !found || action != "ALLOW" {
				ok = false
				warnings = append(warnings, "mtls: first matching IPv4 rule for "+source+" on "+mtls+"/tcp is "+action+" (want ALLOW)")
			}
		}
		// No IPv6 source is trusted for the current mTLS service topology.
		// Reject every IPv6 ALLOW, including narrow source-specific exceptions.
		for _, rule := range rules {
			if rule.IPv6 && ufwRuleMatchesPortTCP(rule, mtls) && rule.Action == "ALLOW" {
				ok = false
				warnings = append(warnings, "mtls: unexpected IPv6 ALLOW from "+rule.From+" for "+mtls+"/tcp")
			}
		}
		// IPv4 mTLS must be reachable only from the configured service CIDRs
		// and loopback. A broad or unrelated source-specific ALLOW is unsafe.
		allowedMTLSSources := []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"}
		for _, rule := range rules {
			if rule.IPv6 || !ufwRuleMatchesPortTCP(rule, mtls) || rule.Action != "ALLOW" {
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
			action, found := firstTCPActionFromSource(rules, lp, "127.0.0.1", false)
			if !found || action != "ALLOW" {
				ok = false
				warnings = append(warnings, "lampac: first matching IPv4 loopback rule for "+lp+"/tcp is "+action+" (want ALLOW)")
			}
			for _, rule := range rules {
				if !ufwRuleMatchesPortTCP(rule, lp) || rule.Action != "ALLOW" {
					continue
				}
				if rule.IPv6 || !ufwSourceContainedBy(rule.From, "127.0.0.1") {
					ok = false
					warnings = append(warnings, "lampac: unexpected ALLOW from "+rule.From+" for "+lp+"/tcp")
				}
			}
		}
	} else {
		requireBoth("8445", "ALLOW", "secondary-wss")
		requireBoth("80", "DENY", "secondary-http")
		for _, port := range []string{"8787", "8788", "8789"} {
			requireBoth(port, "DENY", "secondary-"+port)
		}
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
