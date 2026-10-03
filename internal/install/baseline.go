package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/firewall"
)

// BaselineReport is a machine-readable host baseline integrity check.
type BaselineReport struct {
	RoleFile      string `json:"role_file"`       // primary|secondary|missing
	RoleOK        bool   `json:"role_ok"`
	AptPinPresent bool   `json:"apt_pin_present"`
	AptPinOK      bool   `json:"apt_pin_ok"`
	WatchdogOK    bool   `json:"watchdog_ok"`
	FirewallOK    bool   `json:"firewall_ok"`
	HosterClean   bool   `json:"hoster_clean"`
	OK            bool   `json:"ok"`
	Issues        []string `json:"issues,omitempty"`
}

// CheckHostBaseline verifies role file, apt hoster pin, firewall, watchdog, hoster residual.
func CheckHostBaseline() BaselineReport {
	var r BaselineReport
	if b, err := os.ReadFile("/etc/netductor/role"); err == nil {
		r.RoleFile = strings.TrimSpace(strings.ToLower(string(b)))
		r.RoleOK = r.RoleFile == "primary" || r.RoleFile == "secondary"
	} else {
		r.RoleFile = "missing"
		r.RoleOK = false
		r.Issues = append(r.Issues, "role file missing (/etc/netductor/role)")
	}

	if b, err := os.ReadFile(aptPreferencesHoster); err == nil {
		r.AptPinPresent = true
		s := string(b)
		r.AptPinOK = strings.Contains(s, "Pin-Priority: -1") && strings.Contains(s, "zabbix")
		if !r.AptPinOK {
			r.Issues = append(r.Issues, "apt pin file incomplete")
		}
	} else {
		r.AptPinPresent = false
		r.AptPinOK = false
		r.Issues = append(r.Issues, "apt pin missing ("+aptPreferencesHoster+")")
	}

	if out, err := exec.Command("systemctl", "is-active", "netductor-stack-watchdog.timer").CombinedOutput(); err == nil &&
		strings.TrimSpace(string(out)) == "active" {
		r.WatchdogOK = true
	} else {
		r.WatchdogOK = false
		r.Issues = append(r.Issues, "netductor-stack-watchdog.timer not active")
	}

	role := r.RoleFile
	if role != "primary" && role != "secondary" {
		role = ""
	}
	fw := firewall.Collect(role)
	r.FirewallOK = fw.OK
	if !fw.OK {
		r.Issues = append(r.Issues, fmt.Sprintf("firewall not ok backend=%s", fw.Backend))
	}

	hu, hp := HostAgentFindings()
	pkgs, paths := HostAgentResidual()
	procs := HostAgentProcs()
	r.HosterClean = len(hu) == 0 && len(hp) == 0 && len(pkgs) == 0 && len(paths) == 0 && len(procs) == 0
	if !r.HosterClean {
		r.Issues = append(r.Issues, "hoster residual present (run host-audit --purge)")
	}

	r.OK = r.RoleOK && r.AptPinOK && r.WatchdogOK && r.FirewallOK && r.HosterClean
	return r
}
