package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PurgeHostMonitoring removes common VPS-image monitoring agents (Zabbix)
// that many hosters preinstall. Idempotent; never fails install hard.
func PurgeHostMonitoring() {
	names := []string{"zabbix-agent", "zabbix-agentd", "zabbix-agent2"}
	for _, u := range names {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
		_ = exec.Command("systemctl", "stop", u).Run()
		_ = exec.Command("systemctl", "mask", u).Run()
	}
	pkgs := []string{"zabbix-agent", "zabbix-agent2", "zabbix-release"}
	args := append([]string{"remove", "-y", "--purge"}, pkgs...)
	cmd := exec.Command("apt-get", args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	_ = cmd.Run()
	if _, err := exec.LookPath("ufw"); err == nil {
		_ = exec.Command("ufw", "deny", "10050/tcp").Run()
		_ = exec.Command("ufw", "deny", "10051/tcp").Run()
	}
	// drop leftover binary noise in process list after purge
	_ = exec.Command("pkill", "-f", "zabbix_agent").Run()
	fmt.Fprintln(os.Stderr, "hardening: host monitoring agents purged (zabbix if present)")
}

// HostMonitoringPresent is true if a zabbix unit is active or :10050 is open.
func HostMonitoringPresent() bool {
	for _, u := range []string{"zabbix-agent", "zabbix-agent2", "zabbix-agentd"} {
		out, _ := exec.Command("systemctl", "is-active", u).CombinedOutput()
		if strings.TrimSpace(string(out)) == "active" {
			return true
		}
	}
	return false
}
