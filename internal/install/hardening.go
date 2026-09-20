package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func InstallHardening() error {
	// Minimal: ufw + BBR + SSH key-only. fail2ban only if NETDUCTOR_FAIL2BAN=1.
	pkgs := []string{"ufw", "curl", "ca-certificates"}
	if os.Getenv("NETDUCTOR_FAIL2BAN") == "1" {
		pkgs = append(pkgs, "fail2ban")
	}
	if err := aptInstall(pkgs...); err != nil {
		fmt.Fprintf(os.Stderr, "hardening apt: %v (continuing)\n", err)
	}

	_ = os.WriteFile("/etc/sysctl.d/99-netductor.conf", []byte(
		"net.core.default_qdisc=fq\nnet.ipv4.tcp_congestion_control=bbr\n"), 0o644)
	_ = run("sysctl", "--system")

	if _, err := exec.LookPath("ufw"); err == nil {
		_ = run("ufw", "allow", "OpenSSH")
		_ = run("ufw", "allow", "22/tcp")
		_ = run("ufw", "allow", "443/tcp")
		_ = run("ufw", "allow", "4443/tcp")
		_ = run("ufw", "allow", "4443/udp")
		_ = run("ufw", "allow", "8443/udp")
		// Agent plane: mTLS only (:8789). Plain :8788 must stay closed.
		_ = run("ufw", "delete", "allow", "8788/tcp")
		_ = run("ufw", "deny", "8788/tcp")
		// Until secondary IP is known, allow 8789 from anywhere; provision tightens to from <ip>.
		_ = run("ufw", "allow", "8789/tcp")
		out, _ := runOut("ufw", "status")
		if !strings.Contains(out, "Status: active") {
			_ = run("bash", "-c", "echo y | ufw --force enable")
		}
	}
	if err := EnsureSSHKeyAndHarden(); err != nil {
		fmt.Fprintf(os.Stderr, "ssh harden: %v (continuing)\n", err)
	}
	// Generate agent CA/server/client material at install (not only on first serve).
	if err := ensureMTLSAtInstall(); err != nil {
		fmt.Fprintf(os.Stderr, "mtls ensure: %v (continuing; serve will retry)\n", err)
	}
	fmt.Fprintln(os.Stderr, "hardening: ufw (no 8788, 8789 mTLS) + bbr + ssh key-only + mtls material")
	return nil
}
