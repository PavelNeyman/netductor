package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func runSvcPaths(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, `usage:
  netductor svc-paths status|health|apply
  netductor svc-paths failover status|tick
  netductor svc-paths failover enable|disable
  netductor svc-paths failover set-users-sp on|off
`)
		os.Exit(2)
	}
	switch args[0] {
	case "status", "health":
		r := svcpaths.Status()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		if !r.SP.Up && !r.PS.Up {
			os.Exit(1)
		}
	case "apply":
		msg, err := svcpaths.Apply()
		if msg != "" {
			fmt.Println(msg)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		r := svcpaths.Status()
		fmt.Printf("sp_up=%v ps_up=%v role=%s\n", r.SP.Up, r.PS.Up, r.HostRole)
	case "failover":
		runSvcFailover(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "unknown svc-paths subcommand")
		os.Exit(2)
	}
}

func runSvcFailover(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor svc-paths failover status|tick|enable|disable|set-users-sp on|off")
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "status":
		p := svcpaths.LoadPolicy()
		s := svcpaths.LoadState()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"policy": p, "state": s})
	case "tick":
		s, sum, err := svcpaths.Tick()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(sum)
		_ = s
	case "enable":
		p := svcpaths.LoadPolicy()
		p.Enabled = true
		_ = svcpaths.SavePolicy(p)
		fmt.Println("failover enabled")
	case "disable":
		p := svcpaths.LoadPolicy()
		p.Enabled = false
		_ = svcpaths.SavePolicy(p)
		fmt.Println("failover disabled")
	case "set-users-sp":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "need on|off")
			os.Exit(2)
		}
		on := args[1] == "on" || args[1] == "1" || args[1] == "true"
		p := svcpaths.LoadPolicy()
		p.UsersToSPOnVLESSDown = on
		_ = svcpaths.SavePolicy(p)
		fmt.Printf("users_to_sp_on_vless_down=%v\n", on)
	default:
		// allow threshold tweaks: fail-threshold N
		if args[0] == "fail-threshold" && len(args) > 1 {
			n, _ := strconv.Atoi(args[1])
			p := svcpaths.LoadPolicy()
			if n > 0 {
				p.FailThreshold = n
			}
			_ = svcpaths.SavePolicy(p)
			fmt.Printf("fail_threshold=%d\n", p.FailThreshold)
			return
		}
		fmt.Fprintln(os.Stderr, "unknown failover subcommand")
		os.Exit(2)
	}
}
