package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Hoster/CM units that must not run on netductor nodes (monitoring + remote config mgmt).
// cloud-init, qemu-guest-agent, chrony/systemd-timesyncd are intentionally absent.
var unwantedUnits = []string{
	"zabbix-agent", "zabbix-agentd", "zabbix-agent2",
	"telegraf", "datadog-agent", "datadog-agent-trace", "datadog-agent-process",
	"newrelic-infra", "amazon-cloudwatch-agent", "google-cloud-ops-agent",
	"node_exporter", "prometheus-node-exporter", "collectd", "netdata",
	"salt-minion", "salt-master", "salt-api",
	"puppet", "puppet-agent", "pxp-agent", "mcollective",
	"chef-client", "chef-solo", "cfengine3",
	"landscape-client", "canonical-livepatch",
	"nrpe", "nagios-nrpe-server", "check-mk-agent", "xinetd",
	"snmpd", "snmptrapd",
	"monit", "glances",
	"ossec", "wazuh-agent",
	"besclient", "scaleft-proxy", // rare RMM
	"anydesk", "teamviewerd",
}

var unwantedPackages = []string{
	"zabbix-agent", "zabbix-agent2", "zabbix-release",
	"telegraf", "datadog-agent", "newrelic-infra",
	"prometheus-node-exporter", "collectd", "netdata",
	"salt-minion", "salt-master", "salt-common",
	"puppet-agent", "puppet", "chef", "chef-client",
	"landscape-client", "nagios-nrpe-server", "monitoring-plugins",
	"snmpd", "snmp", "monit",
	"wazuh-agent", "ossec-hids-agent",
}

// Ports often bound by hoster agents (doctor WARN if listening on non-loopback).
var unwantedListenPorts = []string{
	"10050", "10051", // zabbix
	"9100",           // node_exporter
	"9273",           // telegraf prom
	"8125",           // statsd
	"161", "162",     // snmp
	"4505", "4506",   // salt
	"8140",           // puppet
	"5666",           // nrpe
	"19999",          // netdata
	"2812",           // monit
}

// PurgeHostMonitoring removes common VPS-image monitoring + CM agents.
// Idempotent; never fails install hard.
func PurgeHostMonitoring() {
	for _, u := range unwantedUnits {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
		_ = exec.Command("systemctl", "stop", u).Run()
		_ = exec.Command("systemctl", "mask", u).Run()
	}
	args := append([]string{"remove", "-y", "--purge"}, unwantedPackages...)
	cmd := exec.Command("apt-get", args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	_ = cmd.Run()
	if _, err := exec.LookPath("ufw"); err == nil {
		for _, p := range unwantedListenPorts {
			_ = exec.Command("ufw", "deny", p+"/tcp").Run()
			_ = exec.Command("ufw", "deny", p+"/udp").Run()
		}
	}
	_ = exec.Command("pkill", "-f", "zabbix_agent").Run()
	_ = exec.Command("pkill", "-f", "salt-minion").Run()
	fmt.Fprintln(os.Stderr, "hardening: hoster monitoring/CM agents purged (if present)")
}

// HostAgentFindings lists active unwanted units and open unwanted ports.
func HostAgentFindings() (units []string, ports []string) {
	for _, u := range unwantedUnits {
		out, _ := exec.Command("systemctl", "is-active", u).CombinedOutput()
		if strings.TrimSpace(string(out)) == "active" {
			units = append(units, u)
		}
	}
	// ss -tuln fallback
	out, err := exec.Command("ss", "-tuln").CombinedOutput()
	if err != nil {
		out, _ = exec.Command("netstat", "-tuln").CombinedOutput()
	}
	s := string(out)
	for _, p := range unwantedListenPorts {
		// match :PORT with word boundary-ish
		if strings.Contains(s, ":"+p+" ") || strings.Contains(s, ":"+p+"\n") || strings.HasSuffix(strings.TrimSpace(s), ":"+p) {
			// also match :::PORT
			ports = append(ports, p)
			continue
		}
		if strings.Contains(s, ":"+p) {
			ports = append(ports, p)
		}
	}
	return units, ports
}

// HostMonitoringPresent is true if a known agent unit is active or a classic port is open.
func HostMonitoringPresent() bool {
	u, p := HostAgentFindings()
	return len(u) > 0 || len(p) > 0
}
