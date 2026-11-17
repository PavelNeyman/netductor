package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/controlplane"
)

func runControl(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor control status
  netductor control sot-check
  netductor control endpoints
  netductor control set-role active|standby --confirm
  netductor control promote --confirm   # alias: set-role active`)
		os.Exit(2)
	}
	switch args[0] {
	case "status":
		r := controlplane.Get()
		fmt.Printf("control_role=%s write_allowed=%v file=%s\n", r, controlplane.WriteAllowed(), controlplane.RolePath())
		eps := controlplane.ReadEndpoints()
		if len(eps) == 0 {
			fmt.Println("control_endpoints: (none — optional " + controlplane.EndpointsPath() + ")")
		} else {
			fmt.Println("control_endpoints:")
			for _, e := range eps {
				fmt.Println(" ", e)
			}
		}
	case "sot-check", "check-sot":
		ok, lines := controlplane.CheckLocalSoT()
		for _, l := range lines {
			fmt.Println(l)
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "sot-check: FAIL (required paths missing — restore backup before promote)")
			os.Exit(1)
		}
		fmt.Println("sot-check: OK")
	case "endpoints":
		for _, e := range controlplane.ReadEndpoints() {
			fmt.Println(e)
		}
		if len(controlplane.ReadEndpoints()) == 0 {
			fmt.Fprintln(os.Stderr, "# empty; write one URL per line to", controlplane.EndpointsPath())
		}
	case "set-role":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor control set-role active|standby --confirm")
			os.Exit(2)
		}
		role := strings.ToLower(strings.TrimSpace(args[1]))
		confirm := false
		for _, a := range args[2:] {
			if a == "--confirm" || a == "-y" {
				confirm = true
			}
		}
		if !confirm {
			fmt.Fprintln(os.Stderr, "refusing without --confirm (avoids accidental split-brain)")
			os.Exit(2)
		}
		var r controlplane.Role
		switch role {
		case "active", "promote":
			r = controlplane.RoleActive
		case "standby", "passive":
			r = controlplane.RoleStandby
		default:
			fmt.Fprintf(os.Stderr, "unknown role %q\n", role)
			os.Exit(2)
		}
		if err := controlplane.SetRole(r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("control_role=%s write_allowed=%v\n", controlplane.Get(), controlplane.WriteAllowed())
		if r == controlplane.RoleActive {
			fmt.Fprintln(os.Stderr, "NOTE: ensure the previous active node is set to standby or powered off (fencing).")
		}
		if r == controlplane.RoleStandby {
			fmt.Fprintln(os.Stderr, "NOTE: mutating API returns 503 until promote; agents need SERVERS= this host only after promote.")
		}
	case "promote":
		confirm := false
		for _, a := range args[1:] {
			if a == "--confirm" || a == "-y" {
				confirm = true
			}
		}
		if !confirm {
			fmt.Fprintln(os.Stderr, "usage: netductor control promote --confirm")
			os.Exit(2)
		}
		ok, lines := controlplane.CheckLocalSoT()
		for _, l := range lines {
			fmt.Println(l)
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "promote aborted: sot-check FAIL — restore SoT from backup first")
			os.Exit(1)
		}
		if err := controlplane.SetRole(controlplane.RoleActive); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("promoted control_role=%s write_allowed=%v\n", controlplane.Get(), controlplane.WriteAllowed())
		fmt.Fprintln(os.Stderr, "Fence old active: netductor control set-role standby --confirm  (on the old node), or power it off.")
	default:
		fmt.Fprintf(os.Stderr, "unknown control subcommand %q\n", args[0])
		os.Exit(2)
	}
}
