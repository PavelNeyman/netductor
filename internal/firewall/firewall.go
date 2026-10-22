// Package firewall applies role-based host firewall for netductor nodes.
// Preferred backend: ufw (installed if missing). Fallback: iptables-nft or legacy iptables.
// Never silently skip: Apply returns error if no backend can enforce rules.
package firewall

import (
	"fmt"
	"net"
	"os"
	"os/exec"
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
// Live secondary agent wins over a stale role file (stack apply used to force primary).
func DetectRole() string { return detectRole() }

func detectRole() string {
	// Live secondary agent is definitive — prevents wrong role=primary after node upgrade paths.
	if st, _ := exec.Command("systemctl", "is-active", "netductor-secondary-agent").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "secondary"
	}
	// Explicit role file (written by install / firewall apply).
	if b, err := os.ReadFile("/etc/netductor/role"); err == nil {
		s := strings.TrimSpace(strings.ToLower(string(b)))
		if s == "secondary" {
			return "secondary"
		}
		if s == "primary" || s == "core" {
			return "primary"
		}
	}
	// Primary control-plane markers (bundle.json may exist on primary after export — do NOT treat as secondary).
	if _, err := os.Stat("/var/lib/netductor/READY.txt"); err == nil {
		return "primary"
	}
	if st, _ := exec.Command("systemctl", "is-active", "netductor-api").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "primary"
	}
	if st, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "primary"
	}
	// Last resort: secondary bundle only if no primary markers above.
	if _, err := os.Stat("/var/lib/netductor/secondary/bundle.json"); err == nil {
		return "secondary"
	}
	return "primary"
}

// persistRole writes /etc/netductor/role so status matches last apply.
func persistRole(role string) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "primary" && role != "secondary" {
		return
	}
	_ = os.MkdirAll("/etc/netductor", 0o755)
	_ = os.WriteFile("/etc/netductor/role", []byte(role+"\n"), 0o644)
}

func sshPort() string {
	return strconv.Itoa(hardening.SSHPort())
}

func applyUFW(role string) error {
	run := func(args ...string) error {
		cmd := exec.Command("ufw", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := string(out)
			if strings.Contains(msg, "Skipping") || strings.Contains(msg, "Could not delete") {
				return nil
			}
			return fmt.Errorf("ufw %v: %v (%s)", args, err, truncate(msg, 160))
		}
		return nil
	}
	must := func(args ...string) error { return run(args...) }
	if os.Getenv("NETDUCTOR_FW_RESET") == "1" {
		_ = exec.Command("bash", "-c", "echo y | ufw --force reset").Run()
	}
	// H-2: critical policy steps must not be ignored.
	if err := must("default", "deny", "incoming"); err != nil {
		return err
	}
	if err := must("default", "allow", "outgoing"); err != nil {
		return err
	}
	_ = run("default", "deny", "routed")

	sp := sshPort()
	if err := must("allow", sp+"/tcp", "comment", "netductor-ssh"); err != nil {
		return err
	}
	_ = run("deny", "22/tcp", "comment", "netductor-no-ssh22")
	if err := must("allow", "443/tcp", "comment", "netductor-vless"); err != nil {
		return err
	}
	_ = run("deny", "10050/tcp", "comment", "no-zabbix")
	_ = run("deny", "10051/tcp", "comment", "no-zabbix")

	if role == "primary" {
		redir := ndconfig.RedirectHTTPSPort()
		if err := must("allow", redir+"/tcp", "comment", "netductor-redirect"); err != nil {
			return err
		}
		_ = run("allow", redir+"/udp", "comment", "netductor-redirect")
		if err := must("allow", "80/tcp", "comment", "netductor-http-acme"); err != nil {
			return err
		}
		_ = run("allow", "8444/tcp", "comment", "nd-wss-sp")
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
			_ = run("deny", lp+"/tcp", "comment", "lampac-wan-deny")
			_ = run("allow", "from", "127.0.0.1", "to", "any", "port", lp, "proto", "tcp", "comment", "lampac-local")
		}
	} else {
		_ = run("allow", "8445/tcp", "comment", "nd-wss-ps")
		if err := must("deny", "8787/tcp", "comment", "no-api-wan"); err != nil {
			return err
		}
		if err := must("deny", "8788/tcp", "comment", "no-plain-agent"); err != nil {
			return err
		}
		if err := must("deny", "8789/tcp", "comment", "no-mtls-wan-secondary"); err != nil {
			return err
		}
		_ = run("deny", "80/tcp", "comment", "no-http-secondary")
	}

	out, err := exec.Command("bash", "-c", "echo y | ufw --force enable").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw enable: %v (%s)", err, truncate(string(out), 200))
	}
	persistRole(role)
	fmt.Fprintf(os.Stderr, "firewall: ufw active role=%s ssh=%s\n", role, sp)
	return nil
}

func applyIPTables(role string) error {
	// Use filter INPUT with netductor chain for idempotency.
	// Critical rules must succeed — do not report success if chain/jump/ports failed.
	run := func(args ...string) error {
		cmd := exec.Command("iptables", args...)
		out, err := cmd.CombinedOutput()
		msg := string(out)
		if err != nil {
			if strings.Contains(strings.ToLower(msg), "exist") {
				return nil
			}
			// -N when chain exists is ok
			if len(args) > 0 && args[0] == "-N" && strings.Contains(strings.ToLower(msg), "chain") {
				return nil
			}
			return fmt.Errorf("iptables %v: %v (%s)", args, err, truncate(msg, 160))
		}
		return nil
	}
	must := func(args ...string) error {
		return run(args...)
	}
	if err := must("-N", "NETDUCTOR"); err != nil {
		// already exists is fine via run(); other errors abort
		if !strings.Contains(err.Error(), "exist") {
			// try flush only if chain usable
			_ = run("-F", "NETDUCTOR")
		}
	}
	if err := must("-F", "NETDUCTOR"); err != nil {
		return fmt.Errorf("iptables fallback: cannot flush NETDUCTOR: %w", err)
	}
	if exec.Command("iptables", "-C", "INPUT", "-j", "NETDUCTOR").Run() != nil {
		if err := must("-I", "INPUT", "1", "-j", "NETDUCTOR"); err != nil {
			return fmt.Errorf("iptables fallback: INPUT jump: %w", err)
		}
	}
	sp := sshPort()
	allowTCP := func(port string) error {
		return must("-A", "NETDUCTOR", "-p", "tcp", "--dport", port, "-j", "ACCEPT")
	}
	denyTCP := func(port string) error {
		return must("-A", "NETDUCTOR", "-p", "tcp", "--dport", port, "-j", "DROP")
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
			if err := must("-A", "NETDUCTOR", "-p", "tcp", "-s", c, "--dport", mtls, "-j", "ACCEPT"); err != nil {
				return fmt.Errorf("iptables fallback mtls allow %s: %w", c, err)
			}
		}
		if err := must("-A", "NETDUCTOR", "-p", "tcp", "--dport", mtls, "-j", "DROP"); err != nil {
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
	if err := must("-I", "NETDUCTOR", "1", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("iptables fallback established: %w", err)
	}
	if err := must("-I", "NETDUCTOR", "2", "-i", "lo", "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("iptables fallback lo: %w", err)
	}
	// note: do NOT set default DROP on INPUT via iptables alone — risk lockout
	fmt.Fprintf(os.Stderr, "firewall: iptables NETDUCTOR chain applied role=%s (fallback; prefer ufw)\n", role)
	if _, err := exec.LookPath("netfilter-persistent"); err == nil {
		_ = exec.Command("netfilter-persistent", "save").Run()
	} else if _, err := exec.LookPath("iptables-save"); err == nil {
		_ = exec.Command("bash", "-c", "iptables-save > /etc/iptables.rules.netductor").Run()
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
		out, _ := exec.Command("iptables", "-S", "NETDUCTOR").CombinedOutput()
		s := string(out)
		st.Active = strings.Contains(s, "NETDUCTOR") && !strings.Contains(s, "No chain")
		st.Detail = firstLines(s, 20)
		// Do not report OK for iptables fallback — role policy is not fully verified.
		st.OK = false
		if !st.Active || !strings.Contains(s, "--dport "+sshPort()) {
			st.Warnings = append(st.Warnings, "iptables NETDUCTOR chain missing or incomplete (ufw preferred)")
		} else {
			st.Warnings = append(st.Warnings, "iptables fallback: policy not fully verified — not OK (prefer ufw)")
		}
		return st
	}
	st.Backend = BackendNone
	st.OK = false
	st.Detail = "no ufw and no iptables"
	st.Warnings = append(st.Warnings, "CRITICAL: no host firewall backend")
	return st
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
