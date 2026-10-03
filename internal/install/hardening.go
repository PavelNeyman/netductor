package install

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/firewall"
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

	// Host firewall: always try ufw install; iptables fallback inside firewall.ApplyRole.
	if err := firewall.ApplyRole("primary"); err != nil {
		fmt.Fprintf(os.Stderr, "firewall apply primary: %v\n", err)
	}
	_ = ApplyAgentFirewall()
	if err := EnsureSSHKeyAndHarden(); err != nil {
		fmt.Fprintf(os.Stderr, "ssh harden: %v (continuing)\n", err)
	}
	if os.Getenv("NETDUCTOR_FAIL2BAN") != "0" {
		if err := hardening.EnsureFail2banSSH(); err != nil {
			fmt.Fprintf(os.Stderr, "fail2ban: %v (continuing)\n", err)
		}
	}
	PurgeHostMonitoring()
	if err := ensureMTLSAtInstall(); err != nil {
		fmt.Fprintf(os.Stderr, "mtls ensure: %v (continuing; serve will retry)\n", err)
	}
	fmt.Fprintf(os.Stderr, "hardening: firewall + mTLS :8789 + ssh key-only Port %d + fail2ban\n", hardening.SSHPort())
	return nil
}
