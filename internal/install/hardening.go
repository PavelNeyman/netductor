package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/servicenet"
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
	if err := ensureWatchdogTimer(); err != nil {
		fmt.Fprintf(os.Stderr, "watchdog: %v (continuing)\n", err)
	}
	if err := servicenet.Ensure(); err != nil {
		fmt.Fprintf(os.Stderr, "servicenet: %v (continuing)\n", err)
	}
	fmt.Fprintf(os.Stderr, "hardening: firewall + mTLS :8789 + ssh key-only Port %d + fail2ban + watchdog\n", hardening.SSHPort())
	return nil
}

func ensureWatchdogTimer() error {
	out, err := exec.Command("netductor", "stack", "watchdog-install").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// EnsureHostBaseline re-applies role firewall (writes /etc/netductor/role), stack watchdog timer,
// and apt pin block for hoster packages. Idempotent.
func EnsureHostBaseline(role string) error {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		role = "primary"
	}
	if role != "primary" && role != "secondary" {
		return fmt.Errorf("role must be primary|secondary, got %q", role)
	}
	if err := firewall.ApplyRole(role); err != nil {
		return err
	}
	if err := ensureWatchdogTimer(); err != nil {
		fmt.Fprintf(os.Stderr, "watchdog-install: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "host baseline: firewall="+role+" + netductor-stack-watchdog.timer")
	}
	if err := WriteHosterAptBlock(); err != nil {
		fmt.Fprintf(os.Stderr, "hoster apt pin: %v\n", err)
	}
	return nil
}
