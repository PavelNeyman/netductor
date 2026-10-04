// Package firewall applies role-based host firewall for netductor nodes.
// Preferred backend: ufw (installed if missing). Fallback: iptables-nft or legacy iptables.
// Never silently skip: Apply returns error if no backend can enforce rules.
package firewall

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

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

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

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

func detectRole() string {
	// Explicit role file wins (written by install / firewall apply).
	if b, err := os.ReadFile("/etc/netductor/role"); err == nil {
		s := strings.TrimSpace(strings.ToLower(string(b)))
		if s == "secondary" {
			return "secondary"
		}
		if s == "primary" || s == "core" {
			return "primary"
		}
	}
	// Live secondary agent is definitive.
	if st, _ := exec.Command("systemctl", "is-active", "netductor-secondary-agent").CombinedOutput(); strings.TrimSpace(string(st)) == "active" {
		return "secondary"
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
			// delete of missing rule is ok
			msg := string(out)
			if strings.Contains(msg, "Skipping") || strings.Contains(msg, "Could not delete") {
				return nil
			}
			return fmt.Errorf("ufw %v: %v (%s)", args, err, truncate(msg, 160))
		}
		return nil
	}
	// Full reset only when explicitly requested (avoids lockout on live SSH).
	if os.Getenv("NETDUCTOR_FW_RESET") == "1" {
		_ = exec.Command("bash", "-c", "echo y | ufw --force reset").Run()
	}
	_ = run("default", "deny", "incoming")
	_ = run("default", "allow", "outgoing")
	_ = run("default", "deny", "routed")

	sp := sshPort()
	_ = run("allow", sp+"/tcp", "comment", "netductor-ssh")
	_ = run("deny", "22/tcp", "comment", "netductor-no-ssh22")
	_ = run("allow", "443/tcp", "comment", "netductor-vless")
	_ = run("deny", "10050/tcp", "comment", "no-zabbix")
	_ = run("deny", "10051/tcp", "comment", "no-zabbix")

	if role == "primary" {
		redir := ndconfig.RedirectHTTPSPort()
		_ = run("allow", redir+"/tcp", "comment", "netductor-redirect")
		_ = run("allow", redir+"/udp", "comment", "netductor-redirect")
		// ACME http-01 if used
		_ = run("allow", "80/tcp", "comment", "netductor-http-acme")
		// WSS SP server on primary
		_ = run("allow", "8444/tcp", "comment", "nd-wss-sp")
		_ = run("deny", "8788/tcp", "comment", "no-plain-agent")
		// mTLS only from service path CIDRs + localhost
		mtls := ndconfig.AgentMTLSPort()
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"} {
			_ = run("allow", "from", c, "to", "any", "port", mtls, "proto", "tcp", "comment", "netductor-api")
		}
		// Service uplink from secondary only (vless-svc). Not a per-catalog port:
		// lampac/git/registry stay on loopback; clients reach them via VLESS + DNAT.
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR()} {
			_ = run("allow", "from", c, "to", "any", "port", "9443", "proto", "tcp", "comment", "netductor-svc-uplink")
		}
		_ = run("deny", "9443/tcp", "comment", "no-svc-uplink-wan")
		// lampac localhost only
		lp := ndconfig.LampacPort()
		if lp != "" && lp != "0" {
			_ = run("deny", lp+"/tcp", "comment", "lampac-wan-deny")
			_ = run("allow", "from", "127.0.0.1", "to", "any", "port", lp, "proto", "tcp", "comment", "lampac-local")
		}
	} else {
		// secondary: VLESS 443, SSH, WSS PS server
		_ = run("allow", "8445/tcp", "comment", "nd-wss-ps")
		// no control-plane ports on WAN
		_ = run("deny", "8787/tcp", "comment", "no-api-wan")
		_ = run("deny", "8788/tcp", "comment", "no-plain-agent")
		_ = run("deny", "8789/tcp", "comment", "no-mtls-wan-secondary")
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
	// Use filter INPUT with netductor chain for idempotency
	run := func(args ...string) error {
		cmd := exec.Command("iptables", args...)
		out, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(string(out), "No chain") {
			// ignore duplicate rules somewhat
			if strings.Contains(strings.ToLower(string(out)), "exist") {
				return nil
			}
		}
		return nil
	}
	_ = run("-N", "NETDUCTOR")
	_ = run("-F", "NETDUCTOR")
	// jump once
	_ = exec.Command("iptables", "-C", "INPUT", "-j", "NETDUCTOR").Run()
	if exec.Command("iptables", "-C", "INPUT", "-j", "NETDUCTOR").Run() != nil {
		_ = run("-I", "INPUT", "1", "-j", "NETDUCTOR")
	}
	sp := sshPort()
	allowTCP := func(port string) {
		_ = run("-A", "NETDUCTOR", "-p", "tcp", "--dport", port, "-j", "ACCEPT")
	}
	denyTCP := func(port string) {
		_ = run("-A", "NETDUCTOR", "-p", "tcp", "--dport", port, "-j", "DROP")
	}
	allowTCP(sp)
	denyTCP("22")
	allowTCP("443")
	denyTCP("10050")
	denyTCP("10051")
	if role == "primary" {
		redir := ndconfig.RedirectHTTPSPort()
		allowTCP(redir)
		allowTCP("80")
		allowTCP("8444")
		denyTCP("8788")
		mtls := ndconfig.AgentMTLSPort()
		for _, c := range []string{ndconfig.SvcSPCIDR(), ndconfig.SvcPSCIDR(), "127.0.0.1"} {
			_ = run("-A", "NETDUCTOR", "-p", "tcp", "-s", c, "--dport", mtls, "-j", "ACCEPT")
		}
		_ = run("-A", "NETDUCTOR", "-p", "tcp", "--dport", mtls, "-j", "DROP")
	} else {
		allowTCP("8445")
		denyTCP("8787")
		denyTCP("8788")
		denyTCP("8789")
		denyTCP("80")
	}
	// established
	_ = run("-I", "NETDUCTOR", "1", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	_ = run("-I", "NETDUCTOR", "2", "-i", "lo", "-j", "ACCEPT")
	// note: do NOT set default DROP on INPUT via iptables alone — risk lockout on unknown mgmt ports
	// NETDUCTOR chain drops known-bad; hoster may still allow rest until operator tightens
	fmt.Fprintf(os.Stderr, "firewall: iptables NETDUCTOR chain applied role=%s (fallback; prefer ufw)\n", role)
	// persist best-effort
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
		out, _ := exec.Command("ufw", "status").CombinedOutput()
		s := string(out)
		st.Active = strings.Contains(s, "Status: active")
		st.Detail = firstLines(s, 12)
		st.OK = st.Active && strings.Contains(s, sshPort()) && strings.Contains(s, "443")
		if !st.Active {
			st.Warnings = append(st.Warnings, "ufw installed but inactive")
		}
		if !strings.Contains(s, sshPort()) {
			st.Warnings = append(st.Warnings, "ssh port rule missing in ufw status")
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
		st.OK = st.Active && strings.Contains(s, "--dport "+sshPort())
		if !st.OK {
			st.Warnings = append(st.Warnings, "iptables NETDUCTOR chain missing or incomplete (ufw preferred)")
		}
		return st
	}
	st.Backend = BackendNone
	st.OK = false
	st.Detail = "no ufw and no iptables"
	st.Warnings = append(st.Warnings, "CRITICAL: no host firewall backend")
	return st
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
