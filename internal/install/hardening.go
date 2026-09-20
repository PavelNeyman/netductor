package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func InstallHardening() error {
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
		_ = run("ufw", "delete", "allow", "8788/tcp")
		_ = run("ufw", "deny", "8788/tcp")
		_ = ApplyAgentFirewall()
		out, _ := runOut("ufw", "status")
		if !strings.Contains(out, "Status: active") {
			_ = run("bash", "-c", "echo y | ufw --force enable")
		}
	}
	if err := EnsureSSHKeyAndHarden(); err != nil {
		fmt.Fprintf(os.Stderr, "ssh harden: %v (continuing)\n", err)
	}
	if err := ensureMTLSAtInstall(); err != nil {
		fmt.Fprintf(os.Stderr, "mtls ensure: %v (continuing; serve will retry)\n", err)
	}
	fmt.Fprintln(os.Stderr, "hardening: ufw + mTLS :8789 (edge-friendly) + ssh key-only")
	return nil
}
