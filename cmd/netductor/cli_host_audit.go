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
	// extra process hints
	procs := []string{}
	out, _ := exec.Command("ps", "ax", "-o", "comm=").CombinedOutput()
	for _, name := range []string{"zabbix_agentd", "zabbix_agent2", "salt-minion", "telegraf", "datadog-agent", "node_exporter", "ossec-agentd", "wazuh-agentd", "puppet", "chef-client"} {
		if strings.Contains(string(out), name) {
			procs = append(procs, name)
		}
	}
	report := map[string]any{
		"unwanted_units":  units,
		"unwanted_ports":  ports,
		"unwanted_procs":  procs,
		"clean":           len(units) == 0 && len(ports) == 0 && len(procs) == 0,
	}
	if purge {
		install.PurgeHostMonitoring()
		units, ports = install.HostAgentFindings()
		report["after_purge_units"] = units
		report["after_purge_ports"] = ports
		report["purged"] = true
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		fmt.Println("=== netductor host-audit (hoster monitoring / CM) ===")
		if len(units) == 0 {
			fmt.Println("units: (none active)")
		} else {
			fmt.Println("units ACTIVE:", strings.Join(units, ", "))
		}
		if len(ports) == 0 {
			fmt.Println("ports: (none of watched list)")
		} else {
			fmt.Println("ports OPEN:", strings.Join(ports, ", "))
		}
		if len(procs) == 0 {
			fmt.Println("procs: (none of watched list)")
		} else {
			fmt.Println("procs:", strings.Join(procs, ", "))
		}
		if report["clean"].(bool) {
			fmt.Println("result: CLEAN")
		} else {
			fmt.Println("result: FINDINGS — review / run: netductor host-audit --purge")
			os.Exit(1)
		}
	}
}
