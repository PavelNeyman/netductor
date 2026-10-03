package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/install"
)

func runHostAudit(args []string) {
	jsonOut := false
	purge := false
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		}
		if a == "--purge" {
			purge = true
		}
	}
	units, ports := install.HostAgentFindings()
	pkgs, paths := install.HostAgentResidual()
	procs := []string{}
	out, _ := exec.Command("ps", "ax", "-o", "comm=").CombinedOutput()
	for _, name := range []string{"zabbix_agentd", "zabbix_agent2", "salt-minion", "telegraf", "datadog-agent", "node_exporter", "ossec-agentd", "wazuh-agentd", "puppet", "chef-client"} {
		if strings.Contains(string(out), name) {
			procs = append(procs, name)
		}
	}
	clean := len(units) == 0 && len(ports) == 0 && len(procs) == 0 && len(pkgs) == 0 && len(paths) == 0
	report := map[string]any{
		"unwanted_units":    units,
		"unwanted_ports":    ports,
		"unwanted_procs":    procs,
		"unwanted_packages": pkgs,
		"unwanted_paths":    paths,
		"clean":             clean,
	}
	if purge {
		install.PurgeHostMonitoring()
		units, ports = install.HostAgentFindings()
		pkgs, paths = install.HostAgentResidual()
		report["after_purge_units"] = units
		report["after_purge_ports"] = ports
		report["after_purge_packages"] = pkgs
		report["after_purge_paths"] = paths
		report["purged"] = true
		clean = len(units) == 0 && len(ports) == 0 && len(pkgs) == 0 && len(paths) == 0
		report["clean"] = clean
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		if !clean {
			os.Exit(1)
		}
		return
	}
	fmt.Println("=== netductor host-audit (hoster monitoring / CM) ===")
	printList("units ACTIVE", units, "(none active)")
	printList("ports OPEN", ports, "(none of watched list)")
	printList("procs", procs, "(none of watched list)")
	printList("packages", pkgs, "(none of watched list)")
	printList("paths", paths, "(none of watched list)")
	if clean {
		fmt.Println("result: CLEAN")
		return
	}
	fmt.Println("result: FINDINGS — review / run: netductor host-audit --purge")
	os.Exit(1)
}

func printList(label string, items []string, empty string) {
	if len(items) == 0 {
		fmt.Printf("%s: %s\n", label, empty)
		return
	}
	fmt.Printf("%s: %s\n", label, strings.Join(items, ", "))
}
