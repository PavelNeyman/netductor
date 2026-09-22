package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/hardening"
)

func InstallHardening() error {
	pkgs := []string{"ufw", "curl", "wget", "ca-certificates", "fail2ban", "git"}
	if os.Getenv("NETDUCTOR_FAIL2BAN") == "0" {
		pkgs = []string{"ufw", "curl", "wget", "ca-certificates", "git"}
	}
	if err := aptInstall(pkgs...); err != nil {
		fmt.Fprintf(os.Stderr, "hardening apt: %v (continuing)\n", err)
	}

	_ = os.WriteFile("/etc/sysctl.d/99-netductor.conf", []byte(
		"net.core.default_qdisc=fq\nnet.ipv4.tcp_congestion_control=bbr\n"), 0o644)
	_ = run("sysctl", "--system")

	if _, err := exec.LookPath("ufw"); err == nil {
		sshPort := hardening.SSHPort()
		_ = run("ufw", "allow", fmt.Sprintf("%d/tcp", sshPort))
		// keep 22 open during transition so operators are not locked out after Port change
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
	if os.Getenv("NETDUCTOR_FAIL2BAN") != "0" {
		if err := hardening.EnsureFail2banSSH(); err != nil {
			fmt.Fprintf(os.Stderr, "fail2ban: %v (continuing)\n", err)
		}
	}
	if err := ensureMTLSAtInstall(); err != nil {
		fmt.Fprintf(os.Stderr, "mtls ensure: %v (continuing; serve will retry)\n", err)
	}
	fmt.Fprintf(os.Stderr, "hardening: ufw + mTLS :8789 + ssh key-only Port %d + fail2ban\n", hardening.SSHPort())
	return nil
}
