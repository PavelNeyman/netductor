package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/role"
	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/logs"
	"github.com/PavelNeyman/netductor/internal/servicenet"
	"github.com/PavelNeyman/netductor/internal/vpn"
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
		return fmt.Errorf("apply primary firewall policy: %w", err)
	}
	if err := ApplyAgentFirewall(); err != nil {
		return fmt.Errorf("apply agent firewall policy: %w", err)
	}
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
	_ = SnapshotHostBaseline()
	_ = EnsureUnattendedSecurity()
	_ = EnsureBlockyDoT()
	if err := vpn.EnsureCanarySeed(); err != nil {
		fmt.Fprintf(os.Stderr, "canary seed: %v\n", err)
	}
	_ = InstallBackupVerifyTimer()
	_ = InstallAptGuardTimer() // primary install path only
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
func EnsureHostBaseline(roleName string) error {
	roleName = strings.TrimSpace(strings.ToLower(roleName))
	if roleName == "" {
		roleName = "primary"
	}
	if !role.AcceptsRoleFile(roleName) {
		return fmt.Errorf("role must be primary|secondary|control|egress, got %q", roleName)
	}
	diskRole := role.LegacyWrite(roleName) // on-disk dual-write still primary|secondary
	// CF-4: single owner of secondary SSH UFW transition.
	// - Factory path (:22 still listening): bootstrap profile keeps temporary ALLOW :22.
	// - Hardened path (key-only :52222, no :22 listener): full role policy with DENY :22
	//   and removal of netductor-ssh-bootstrap leftovers.
	// Do not reopen :22 on already-hardened secondaries via baseline.
	if diskRole == "secondary" {
		if secondarySSHReadyHardened() {
			if err := firewall.ApplyRole(diskRole); err != nil {
				return fmt.Errorf("apply secondary hardened firewall policy: %w", err)
			}
		} else {
			if err := firewall.ApplyRoleBootstrap(diskRole); err != nil {
				return fmt.Errorf("apply secondary bootstrap firewall policy: %w", err)
			}
		}
	} else if err := firewall.ApplyRole(diskRole); err != nil {
		return err
	}
	if err := ensureWatchdogTimer(); err != nil {
		return fmt.Errorf("watchdog baseline: %w", err)
	}
	fmt.Fprintln(os.Stderr, "host baseline: firewall="+diskRole+" + netductor-stack-watchdog.timer")
	if err := logs.EnsureTimer(); err != nil {
		fmt.Fprintf(os.Stderr, "logs-rotate timer: %v\n", err)
	}
	if err := WriteHosterAptBlock(); err != nil {
		return fmt.Errorf("hoster apt pin baseline: %w", err)
	}
	if err := SnapshotHostBaseline(); err != nil {
		fmt.Fprintf(os.Stderr, "host baseline snapshot: %v\n", err)
	}
	if diskRole == "primary" {
		if err := servicenet.Ensure(); err != nil {
			fmt.Fprintf(os.Stderr, "servicenet: %v\n", err)
		}
		_ = vpn.EnsureCanarySeed()
		_ = InstallBackupVerifyTimer()
	}
	if err := InstallAptGuardTimer(); err != nil {
		fmt.Fprintf(os.Stderr, "aptguard timer: %v\n", err)
	}
	for _, n := range RepairStateOnUpgrade() {
		fmt.Fprintln(os.Stderr, "state repair:", n)
	}
	return nil
}

// secondarySSHReadyHardened is true when the host already runs key-only SSH on the
// netductor port and factory :22 is no longer listening. Baseline must not re-open :22.
func secondarySSHReadyHardened() bool {
	const hardenedPort = uint16(52222)
	listen22, err22 := firewall.TCPPortListening(22)
	if err22 != nil {
		return false
	}
	listenH, errH := firewall.TCPPortListening(hardenedPort)
	if errH != nil {
		return false
	}
	return listenH && !listen22
}
