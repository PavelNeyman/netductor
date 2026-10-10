package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EnsureUnattendedSecurity installs unattended-upgrades for security origin only (no auto-reboot).
func EnsureUnattendedSecurity() error {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return nil
	}
	_ = exec.Command("apt-get", "install", "-y", "unattended-upgrades").Run()
	cfg := `/etc/apt/apt.conf.d/52netductor-unattended`
	body := `Unattended-Upgrade::Allowed-Origins {
	"${distro_id}:${distro_codename}-security";
	"${distro_id}ESMApps:${distro_codename}-apps-security";
	"${distro_id}ESM:${distro_codename}-infra-security";
};
Unattended-Upgrade::Remove-Unused-Kernel-Packages "true";
Unattended-Upgrade::Automatic-Reboot "false";
Unattended-Upgrade::Mail "";
`
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		return err
	}
	// enable timer if present
	_ = exec.Command("systemctl", "enable", "--now", "unattended-upgrades").Run()
	fmt.Fprintln(os.Stderr, "unattended-upgrades: security only, Automatic-Reboot=false")
	return nil
}

// RebootRequired reports whether /var/run/reboot-required exists.
func RebootRequired() (bool, string) {
	b, err := os.ReadFile("/var/run/reboot-required")
	if err != nil {
		return false, ""
	}
	pkgs := ""
	if p, err := os.ReadFile("/var/run/reboot-required.pkgs"); err == nil {
		pkgs = strings.TrimSpace(string(p))
	}
	return true, strings.TrimSpace(string(b)) + " " + pkgs
}
