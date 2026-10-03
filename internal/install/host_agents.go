package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	"otelcol", "otelcol-contrib", "filebeat", "elastic-agent",
	"avahi-daemon", "cups", "cups-browsed", "rpcbind",
}

// Exact package names for apt purge (plus pattern scan for hoster variants).
var unwantedPackages = []string{
	"zabbix-agent", "zabbix-agent2", "zabbix-release", "zabbix-agent-timeweb",
	"telegraf", "datadog-agent", "newrelic-infra",
	"prometheus-node-exporter", "collectd", "netdata",
	"salt-minion", "salt-master", "salt-common",
	"puppet-agent", "puppet", "chef", "chef-client",
	"landscape-client", "nagios-nrpe-server", "monitoring-plugins",
	"snmpd", "snmp", "monit",
	"wazuh-agent", "ossec-hids-agent",
	"avahi-daemon", "cups", "rpcbind",
	"amazon-cloudwatch-agent",
}

// Substrings matched against `dpkg -l` package names (catches hoster forks).
var unwantedPackageSubstr = []string{
	"zabbix", "telegraf", "datadog", "newrelic", "salt-minion", "salt-master",
	"puppet", "chef-client", "node-exporter", "node_exporter", "netdata",
	"collectd", "nrpe", "wazuh", "ossec", "cloudwatch-agent", "otelcol",
	"filebeat", "elastic-agent", "landscape-client",
}

// Residual paths that indicate a prior hoster agent install.
var unwantedPaths = []string{
	"/etc/zabbix", "/opt/zabbix", "/var/log/zabbix",
	"/etc/telegraf", "/etc/datadog-agent", "/opt/datadog-agent",
	"/etc/salt", "/etc/puppet", "/etc/puppetlabs", "/etc/chef",
	"/etc/ossec", "/var/ossec",
	"/opt/splunkforwarder", "/etc/newrelic-infra",
	"/opt/tacticalrmm", "/opt/meshagent",
}

// APT source/keyring residuals from hoster agent installers (e.g. Timeweb zabbix).
var unwantedAptGlobs = []string{
	"/etc/apt/sources.list.d/*zabbix*",
	"/etc/apt/sources.list.d/*timeweb*",
	"/etc/apt/sources.list.d/*telegraf*",
	"/etc/apt/sources.list.d/*datadog*",
	"/etc/apt/sources.list.d/*salt*",
	"/etc/apt/keyrings/*zabbix*",
	"/etc/apt/keyrings/*timeweb*",
	"/etc/apt/trusted.gpg.d/*zabbix*",
	"/etc/apt/trusted.gpg.d/*timeweb*",
}

// Ports often bound by hoster agents (doctor WARN if listening on non-loopback).
var unwantedListenPorts = []string{
	"10050", "10051", // zabbix
	"9100",       // node_exporter
	"9273",       // telegraf prom
	"8125",       // statsd
	"161", "162", // snmp
	"4505", "4506", // salt
	"8140",  // puppet
	"5666",  // nrpe
	"19999", // netdata
	"2812",  // monit
	"5353",  // avahi
	"631",   // cups
}

// PurgeHostMonitoring removes common VPS-image monitoring + CM agents.
// Idempotent; never fails install hard.
func PurgeHostMonitoring() {
	for _, u := range unwantedUnits {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
		_ = exec.Command("systemctl", "stop", u).Run()
		_ = exec.Command("systemctl", "mask", u).Run()
	}
	// Exact list + any dpkg names matching hoster substrings (e.g. zabbix-agent-timeweb).
	pkgs := append([]string{}, unwantedPackages...)
	pkgs = append(pkgs, scanInstalledUnwantedPackages()...)
	pkgs = uniqueStrings(pkgs)
	if len(pkgs) > 0 {
		args := append([]string{"remove", "-y", "--purge"}, pkgs...)
		cmd := exec.Command("apt-get", args...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		_ = cmd.Run()
	}
	if _, err := exec.LookPath("ufw"); err == nil {
		for _, p := range unwantedListenPorts {
			_ = exec.Command("ufw", "deny", p+"/tcp").Run()
			_ = exec.Command("ufw", "deny", p+"/udp").Run()
		}
	}
	_ = exec.Command("pkill", "-f", "zabbix_agent").Run()
	_ = exec.Command("pkill", "-f", "salt-minion").Run()
	// Residual config dirs after apt purge of hoster-custom packages.
	for _, d := range unwantedPaths {
		_ = os.RemoveAll(d)
	}
	for _, f := range scanUnwantedAptResiduals() {
		_ = os.Remove(f)
	}
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
	out, err := exec.Command("ss", "-tuln").CombinedOutput()
	if err != nil {
		out, _ = exec.Command("netstat", "-tuln").CombinedOutput()
	}
	s := string(out)
	for _, p := range unwantedListenPorts {
		if strings.Contains(s, ":"+p+" ") || strings.Contains(s, ":"+p+"\n") || strings.HasSuffix(strings.TrimSpace(s), ":"+p) {
			ports = append(ports, p)
			continue
		}
		if strings.Contains(s, ":"+p) {
			ports = append(ports, p)
		}
	}
	return units, ports
}

// HostAgentResidual reports packages and paths that indicate hoster agents
// even when units are stopped (common after partial purge).
func HostAgentResidual() (packages []string, paths []string) {
	packages = scanInstalledUnwantedPackages()
	for _, d := range unwantedPaths {
		if _, err := os.Stat(d); err == nil {
			paths = append(paths, d)
		}
	}
	paths = append(paths, scanUnwantedAptResiduals()...)
	return packages, uniqueStrings(paths)
}

func scanUnwantedAptResiduals() []string {
	var found []string
	for _, g := range unwantedAptGlobs {
		matches, _ := filepath.Glob(g)
		for _, m := range matches {
			if st, err := os.Stat(m); err == nil && !st.IsDir() {
				found = append(found, m)
			}
		}
	}
	return found
}

// HostMonitoringPresent is true if a known agent unit is active, a classic port is open,
// or residual hoster packages/paths remain.
func HostMonitoringPresent() bool {
	u, p := HostAgentFindings()
	pkgs, paths := HostAgentResidual()
	return len(u) > 0 || len(p) > 0 || len(pkgs) > 0 || len(paths) > 0
}

func scanInstalledUnwantedPackages() []string {
	out, err := exec.Command("dpkg-query", "-W", "-f", "${Package}\t${Status}\n").CombinedOutput()
	if err != nil {
		// Fallback: dpkg -l
		out, err = exec.Command("dpkg", "-l").CombinedOutput()
		if err != nil {
			return nil
		}
	}
	var found []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// dpkg-query: name\tinstall ok installed
		// dpkg -l: ii  name ...
		var name string
		if strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", 2)
			name = parts[0]
			if len(parts) > 1 && !strings.Contains(parts[1], "installed") {
				continue
			}
		} else if strings.HasPrefix(line, "ii ") || strings.HasPrefix(line, "ii\t") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			name = fields[1]
		} else {
			continue
		}
		low := strings.ToLower(name)
		for _, sub := range unwantedPackageSubstr {
			if strings.Contains(low, sub) {
				found = append(found, name)
				break
			}
		}
	}
	return uniqueStrings(found)
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
