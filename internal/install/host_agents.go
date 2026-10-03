package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	"besclient", "scaleft-proxy",
	"anydesk", "teamviewerd",
	"otelcol", "otelcol-contrib", "filebeat", "elastic-agent",
	"avahi-daemon", "cups", "cups-browsed", "rpcbind",
}

// Exact package names preferred for apt purge when installed.
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

// Substrings matched against installed package names (hoster forks).
// Prefer anchors: avoid bare "puppet" matching unrelated names.
var unwantedPackageSubstr = []string{
	"zabbix", "telegraf", "datadog", "newrelic",
	"salt-minion", "salt-master", "salt-common", "salt-api",
	"puppet-agent", "puppet-common", "pxp-agent",
	"chef-client", "chef-solo",
	"node-exporter", "node_exporter", "prometheus-node-exporter",
	"netdata", "collectd", "nrpe", "nagios-nrpe",
	"wazuh", "ossec", "cloudwatch-agent", "otelcol",
	"filebeat", "elastic-agent", "landscape-client",
	"zabbix-agent-timeweb",
}

// packageNameUnwanted reports whether an installed dpkg name looks like a hoster agent.
func packageNameUnwanted(name string) bool {
	low := strings.ToLower(strings.TrimSpace(name))
	if low == "" {
		return false
	}
	if low == "puppet" || strings.HasPrefix(low, "puppet-") {
		// allow library-style names only if clearly agent/cm
		if low == "puppet" || strings.Contains(low, "puppet-agent") ||
			strings.Contains(low, "puppet-common") || strings.Contains(low, "puppetlabs") {
			return true
		}
		return false
	}
	for _, sub := range unwantedPackageSubstr {
		if strings.Contains(low, sub) {
			return true
		}
	}
	return false
}

var unwantedPaths = []string{
	"/etc/zabbix", "/opt/zabbix", "/var/log/zabbix",
	"/etc/telegraf", "/etc/datadog-agent", "/opt/datadog-agent",
	"/etc/salt", "/etc/puppet", "/etc/puppetlabs", "/etc/chef",
	"/etc/ossec", "/var/ossec",
	"/opt/splunkforwarder", "/etc/newrelic-infra",
	"/opt/tacticalrmm", "/opt/meshagent",
}

var unwantedAptGlobs = []string{
	"/etc/apt/sources.list.d/*zabbix*",
	"/etc/apt/sources.list.d/*timeweb*",
	"/etc/apt/sources.list.d/*salt*",
	"/etc/apt/keyrings/*zabbix*",
	"/etc/apt/keyrings/*timeweb*",
	"/etc/apt/trusted.gpg.d/*zabbix*",
	"/etc/apt/trusted.gpg.d/*timeweb*",
}

var unwantedListenPorts = []string{
	"10050", "10051",
	"9100",
	"9273",
	"8125",
	"161", "162",
	"4505", "4506",
	"8140",
	"5666",
	"19999",
	"2812",
	"5353",
	"631",
}

// watchedProcNames exact process names (ps -o comm= is basename-limited).
var watchedProcNames = []string{
	"zabbix_agentd", "zabbix_agent2", "salt-minion", "telegraf",
	"datadog-agent", "node_exporter", "ossec-agentd", "wazuh-agentd",
	"puppet", "chef-client",
}

// aptPinPackages: preferences.d entries with Pin-Priority -1 (block install/upgrade).
var aptPinPackages = []string{
	"zabbix-*",
	"zabbix-agent-timeweb",
	"telegraf",
	"datadog-agent",
	"newrelic-infra",
	"prometheus-node-exporter",
	"collectd",
	"netdata",
	"salt-minion",
	"salt-master",
	"salt-common",
	"salt-api",
	"puppet",
	"puppet-agent",
	"puppet-common",
	"chef",
	"chef-client",
	"landscape-client",
	"nagios-nrpe-server",
	"wazuh-agent",
	"ossec-hids-agent",
	"amazon-cloudwatch-agent",
	"filebeat",
	"elastic-agent",
}

const aptPreferencesHoster = "/etc/apt/preferences.d/netductor-block-hoster"

// PurgeHostMonitoring removes common VPS-image monitoring + CM agents.
// Idempotent; never fails install hard. Only purges packages that are installed.
func PurgeHostMonitoring() {
	for _, u := range unwantedUnits {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
		_ = exec.Command("systemctl", "stop", u).Run()
		_ = exec.Command("systemctl", "mask", u).Run()
	}

	pkgs := scanInstalledUnwantedPackages()
	// Also intersect exact list against installed (scan already covers substrings).
	if len(pkgs) > 0 {
		args := append([]string{"remove", "-y", "--purge"}, pkgs...)
		cmd := exec.Command("apt-get", args...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "hoster apt purge: %v (%s)\n", err, strings.TrimSpace(string(out)))
		} else if len(out) > 0 {
			fmt.Fprintf(os.Stderr, "hoster apt purge: removed %d package(s)\n", len(pkgs))
		}
	}

	// Prefer role firewall deny list; do not spam ufw with every classic port here.

	// Stop residual processes by exact name (not pkill -f).
	for _, n := range watchedProcNames {
		_ = exec.Command("killall", "-q", n).Run()
	}

	for _, d := range unwantedPaths {
		_ = os.RemoveAll(d)
	}
	for _, f := range scanUnwantedAptResiduals() {
		_ = os.Remove(f)
	}

	if err := WriteHosterAptBlock(); err != nil {
		fmt.Fprintf(os.Stderr, "hoster apt pin: %v\n", err)
	}

	fmt.Fprintln(os.Stderr, "hardening: hoster monitoring/CM agents purged (if present)")
}

// WriteHosterAptBlock installs apt preferences that refuse hoster agent packages.
// Safer than apt-mark hold for packages not currently installed.
func WriteHosterAptBlock() error {
	var b strings.Builder
	b.WriteString("# Managed by netductor — do not edit; hoster monitoring/CM must not reinstall.\n")
	b.WriteString("# Remove this file only if you intentionally install these agents.\n")
	for _, pkg := range aptPinPackages {
		b.WriteString("Package: ")
		b.WriteString(pkg)
		b.WriteString("\nPin: release *\nPin-Priority: -1\n\n")
	}
	if err := os.MkdirAll("/etc/apt/preferences.d", 0o755); err != nil {
		return err
	}
	return os.WriteFile(aptPreferencesHoster, []byte(b.String()), 0o644)
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
	ports = listenPortsFromSS(string(out), unwantedListenPorts)
	return units, ports
}

// listenPortsFromSS matches :PORT only as a discrete ss/netstat local port (not substring of larger numbers).
func listenPortsFromSS(s string, want []string) []string {
	// :10050 or :10050\n or *:10050 or 0.0.0.0:10050 — not :100500
	var found []string
	for _, p := range want {
		re := regexp.MustCompile(`:` + regexp.QuoteMeta(p) + `([^0-9]|$)`)
		if re.MatchString(s) {
			found = append(found, p)
		}
	}
	return found
}

// HostAgentProcs lists watched process names currently running.
func HostAgentProcs() []string {
	out, err := exec.Command("ps", "ax", "-o", "comm=").CombinedOutput()
	if err != nil {
		return nil
	}
	// Build set of running comm names
	running := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		n := strings.TrimSpace(line)
		if n != "" {
			running[n] = struct{}{}
		}
	}
	var procs []string
	for _, name := range watchedProcNames {
		if _, ok := running[name]; ok {
			procs = append(procs, name)
		}
	}
	return procs
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
// residual packages/paths remain, or a watched process is running.
func HostMonitoringPresent() bool {
	u, p := HostAgentFindings()
	pkgs, paths := HostAgentResidual()
	procs := HostAgentProcs()
	return len(u) > 0 || len(p) > 0 || len(pkgs) > 0 || len(paths) > 0 || len(procs) > 0
}

func scanInstalledUnwantedPackages() []string {
	out, err := exec.Command("dpkg-query", "-W", "-f", "${Package}\t${Status}\n").CombinedOutput()
	if err != nil {
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
		if packageNameUnwanted(name) {
			found = append(found, name)
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
