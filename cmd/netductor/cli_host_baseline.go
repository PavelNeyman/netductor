package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/install"
)

func runHostBaseline(args []string) {
	if len(args) < 1 {
		fmt.Print(`netductor host-baseline status [--json]
netductor host-baseline apply [primary|secondary]
`)
		return
	}
	switch args[0] {
	case "status":
		r := install.CheckHostBaseline()
		if len(args) > 1 && args[1] == "--json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(r)
			if !r.OK {
				os.Exit(1)
			}
			return
		}
		fmt.Printf("role_file=%s ok=%v\n", r.RoleFile, r.RoleOK)
		fmt.Printf("apt_pin present=%v ok=%v\n", r.AptPinPresent, r.AptPinOK)
		fmt.Printf("firewall_ok=%v watchdog_ok=%v hoster_clean=%v\n", r.FirewallOK, r.WatchdogOK, r.HosterClean)
		fmt.Printf("baseline_ok=%v\n", r.OK)
		for _, i := range r.Issues {
			fmt.Println("ISSUE:", i)
		}
		if !r.OK {
			os.Exit(1)
		}
	case "apply":
		role := "primary"
		if len(args) > 1 {
			role = args[1]
		}
		if err := install.EnsureHostBaseline(role); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// purge residual soft
		install.PurgeHostMonitoring()
		r := install.CheckHostBaseline()
		fmt.Printf("host-baseline applied role=%s ok=%v\n", role, r.OK)
		for _, i := range r.Issues {
			fmt.Println("ISSUE:", i)
		}
		if !r.OK {
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown host-baseline subcommand")
		os.Exit(2)
	}
}
