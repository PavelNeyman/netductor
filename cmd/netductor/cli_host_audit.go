package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/install"
)

func runHostAudit(args []string) {
	jsonOut := false
	purge := false
	dryRun := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--purge":
			purge = true
		case "--dry-run":
			dryRun = true
		}
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "host-audit: root required for accurate results")
	}

	units, ports := install.HostAgentFindings()
	pkgs, paths := install.HostAgentResidual()
	procs := install.HostAgentProcs()
	clean := len(units) == 0 && len(ports) == 0 && len(procs) == 0 && len(pkgs) == 0 && len(paths) == 0

	report := map[string]any{
		"unwanted_units":    units,
		"unwanted_ports":    ports,
		"unwanted_procs":    procs,
		"unwanted_packages": pkgs,
		"unwanted_paths":    paths,
		"clean":             clean,
	}

	if dryRun {
		report["dry_run"] = true
		report["would_purge_packages"] = pkgs
		report["would_remove_paths"] = paths
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(report)
		} else {
			fmt.Println("=== netductor host-audit --dry-run ===")
			printList("would purge packages", pkgs, "(none)")
			printList("would remove paths", paths, "(none)")
			printList("active units", units, "(none)")
			printList("procs", procs, "(none)")
			if clean {
				fmt.Println("result: CLEAN (nothing to purge)")
			} else {
				fmt.Println("result: WOULD PURGE — run: netductor host-audit --purge")
			}
		}
		if !clean {
			os.Exit(1)
		}
		return
	}

	if purge {
		if len(pkgs) > 0 || len(paths) > 0 || len(units) > 0 || len(procs) > 0 {
			fmt.Fprintf(os.Stderr, "host-audit purge: packages=%v paths=%v units=%v procs=%v\n", pkgs, paths, units, procs)
		}
		install.PurgeHostMonitoring()
		units, ports = install.HostAgentFindings()
		pkgs, paths = install.HostAgentResidual()
		procs = install.HostAgentProcs()
		report["after_purge_units"] = units
		report["after_purge_ports"] = ports
		report["after_purge_packages"] = pkgs
		report["after_purge_paths"] = paths
		report["after_purge_procs"] = procs
		report["purged"] = true
		clean = len(units) == 0 && len(ports) == 0 && len(procs) == 0 && len(pkgs) == 0 && len(paths) == 0
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
	fmt.Println("hint: netductor host-audit --dry-run  # list packages/paths without removing")
	os.Exit(1)
}

func printList(label string, items []string, empty string) {
	if len(items) == 0 {
		fmt.Printf("%s: %s\n", label, empty)
		return
	}
	fmt.Printf("%s: %s\n", label, strings.Join(items, ", "))
}
