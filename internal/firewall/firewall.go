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

// ApplyRole installs backend if needed and applies primary or secondary profile.
func ApplyRole(role string) error {
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

func applyUFW(role string) error {
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
	// UFW applies rule changes immediately when already enabled; setting the
	// default incoming policy first can lock out SSH if the allow rule then fails.
	if err := must("allow", sp+"/tcp", "comment", "netductor-ssh"); err != nil {
		return fmt.Errorf("install SSH allow rule before default deny: %w", err)
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
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), "default:") {
			continue
		}
		for _, item := range strings.Split(strings.TrimSpace(strings.TrimPrefix(line, line[:len("Default:")])), ",") {
			item = strings.TrimSpace(item)
			if strings.Contains(strings.ToLower(item), "(incoming)") {
				fields := strings.Fields(item)
				if len(fields) < 2 {
					return false, "unparseable default incoming policy: " + item
				}
				policy := strings.ToLower(fields[0])
				if policy == "deny" || policy == "reject" {
					return true, ""
				}
				return false, "default incoming policy is " + policy + " (must be deny or reject)"
			}
		}
		return false, "default incoming policy is missing from UFW verbose status"
	}
	return false, "UFW verbose status has no Default policy line"
}

// ufwRule is one line from `ufw status numbered` (IPv4 preferred; v6 skipped for policy OK).
type ufwRule struct {
	Num    int
	Port   string // e.g. 22 or 52222
	Proto  string // tcp|udp|""
	Action string // ALLOW|DENY|REJECT|LIMIT
	From   string // Anywhere, 10.87.10.0/30, 127.0.0.1, ...
	Raw    string
}

// parseUfwNumbered parses lines like: [ 1] 52222/tcp    ALLOW IN    Anywhere
func parseUfwNumbered(status string) []ufwRule {
	var rules []ufwRule
	for _, ln := range strings.Split(status, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || !strings.HasPrefix(ln, "[") {
			continue
		}
		// strip [ N]
		rb := strings.Index(ln, "]")
		if rb < 0 {
			continue
		}
		numStr := strings.TrimSpace(ln[1:rb])
		num := 0
		for _, c := range numStr {
			if c >= '0' && c <= '9' {
				num = num*10 + int(c-'0')
			}
		}
		rest := strings.TrimSpace(ln[rb+1:])
		if strings.Contains(rest, "(v6)") {
			continue // evaluate IPv4 path; v6 mirrored separately
		}
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
		// find action
		actionIdx := -1
		for i, f := range fields {
			if f == "ALLOW" || f == "DENY" || f == "REJECT" || f == "LIMIT" {
				actionIdx = i
				break
			}
		}
		if actionIdx < 0 {
			continue
		}
		action := fields[actionIdx]
		// skip IN/OUT
		fromParts := fields[actionIdx+1:]
		for len(fromParts) > 0 && (fromParts[0] == "IN" || fromParts[0] == "OUT") {
			fromParts = fromParts[1:]
		}
		from := strings.Join(fromParts, " ")
		if from == "" {
			from = "Anywhere"
		}
		rules = append(rules, ufwRule{Num: num, Port: port, Proto: proto, Action: action, From: from, Raw: ln})
	}
	return rules
}

func ufwRuleMatchesPortTCP(r ufwRule, port string) bool {
	if r.Proto == "udp" {
		return false
	}
	rp := strings.TrimSpace(r.Port)
	// Global allow/deny (Anywhere, *, empty) applies to every TCP port.
	if rp == "" || rp == "Anywhere" || rp == "Anywhere (v6)" || rp == "*" ||
		strings.EqualFold(rp, "any") {
		return true
	}
	// Port ranges: 8000:9000 or 8000-9000
	for _, sep := range []string{":", "-"} {
		if i := strings.Index(rp, sep); i > 0 {
			lo, hi := rp[:i], rp[i+len(sep):]
			if j := strings.Index(lo, "/"); j >= 0 {
				lo = lo[:j]
			}
			if j := strings.Index(hi, "/"); j >= 0 {
				hi = hi[:j]
			}
			a, ea := strconv.Atoi(lo)
			b, eb := strconv.Atoi(hi)
			pn, ep := strconv.Atoi(port)
			if ea == nil && eb == nil && ep == nil && pn >= a && pn <= b {
				return true
			}
		}
	}
	base := rp
	if j := strings.Index(base, "/"); j >= 0 {
		base = base[:j]
	}
	return base == port
}
func ufwFromIsWAN(from string) bool {
	from = strings.TrimSpace(from)
	return from == "" || from == "Anywhere" || from == "Anywhere (v6)" ||
		strings.EqualFold(from, "any")
}

func ufwFromIsTrustedMTLS(from string) bool {
	from = strings.TrimSpace(from)
	if from == "127.0.0.1" || from == "lo" || strings.HasPrefix(from, "127.0.0.1/") {
		return true
	}
	// Exact match against configured service-path CIDRs (not substring "10.87").
	for _, w := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
		if cidrEqual(from, w) {
			return true
		}
	}
	return false
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

// firstTCPAction walks rules in order; returns first action that applies to port from a WAN source.
func firstTCPActionFromWAN(rules []ufwRule, port string) (action string, found bool) {
	for _, r := range rules {
		if !ufwRuleMatchesPortTCP(r, port) {
			continue
		}
		if !ufwFromIsWAN(r.From) {
			continue // restricted source — not the WAN path
		}
		return r.Action, true
	}
	return "", false
}

// ufwEffectivePolicy uses rule order from `ufw status numbered` (first match wins).
func ufwEffectivePolicy(status, role string) (ok bool, warnings []string) {
	ok = true
	// Default incoming must be deny/reject (not allow).
	low := strings.ToLower(status)
	if strings.Contains(low, "default: allow (incoming)") ||
		strings.Contains(low, "default: allow (incoming,") {
		ok = false
		warnings = append(warnings, "ufw default incoming is ALLOW — must be deny")
	}
	rules := parseUfwNumbered(status)
	// Fallback: non-numbered status still works via order of appearance.
	if len(rules) == 0 {
		for i, ln := range strings.Split(status, "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" || strings.HasPrefix(ln, "Status:") || strings.HasPrefix(ln, "To ") {
				continue
			}
			fields := strings.Fields(ln)
			if len(fields) < 3 {
				continue
			}
			portProto := fields[0]
			port, proto := portProto, ""
			if j := strings.Index(portProto, "/"); j >= 0 {
				port, proto = portProto[:j], portProto[j+1:]
			}
			action := ""
			for _, f := range fields {
				if f == "ALLOW" || f == "DENY" || f == "REJECT" || f == "LIMIT" {
					action = f
					break
				}
			}
			if action == "" || proto == "udp" {
				continue
			}
			from := "Anywhere"
			if ai := -1; true {
				for i, f := range fields {
					if f == action {
						ai = i
						break
					}
				}
				if ai >= 0 && ai+1 < len(fields) {
					from = strings.Join(fields[ai+1:], " ")
					from = strings.TrimPrefix(from, "IN ")
					from = strings.TrimPrefix(from, "OUT ")
				}
			}
			rules = append(rules, ufwRule{Num: i + 1, Port: port, Proto: proto, Action: action, From: from, Raw: ln})
		}
	}

	sp := sshPort()
	requireWAN := func(port, wantAction, label string) {
		act, found := firstTCPActionFromWAN(rules, port)
		if !found {
			ok = false
			warnings = append(warnings, label+": no "+port+"/tcp rule for WAN")
			return
		}
		if act != wantAction {
			ok = false
			warnings = append(warnings, label+": first WAN rule for "+port+"/tcp is "+act+" (want "+wantAction+")")
		}
	}
	requireWAN(sp, "ALLOW", "ssh")
	requireWAN("443", "ALLOW", "vless")
	requireWAN("22", "DENY", "ssh22")
	requireWAN("10050", "DENY", "zabbix")

	if role == "primary" {
		requireWAN("8788", "DENY", "plain-agent")
		mtls := ndconfig.AgentMTLSPort()
		var trustedAllow bool
		var wanAct string
		var wanFound bool
		for _, r := range rules {
			if !ufwRuleMatchesPortTCP(r, mtls) {
				continue
			}
			if ufwFromIsTrustedMTLS(r.From) && r.Action == "ALLOW" {
				trustedAllow = true
			}
			if ufwFromIsWAN(r.From) && !wanFound {
				wanAct, wanFound = r.Action, true
			}
		}
		if !trustedAllow {
			ok = false
			warnings = append(warnings, "mtls: no ALLOW from 10.87/lo for "+mtls+"/tcp")
		}
		if wanFound && wanAct == "ALLOW" {
			ok = false
			warnings = append(warnings, "mtls: first WAN rule for "+mtls+"/tcp is ALLOW (must not be open)")
		}
	} else {
		for _, d := range []string{"8787", "8788", "8789"} {
			requireWAN(d, "DENY", "secondary-"+d)
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
