package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

func runRecovery(args []string) {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, `usage:
  netductor recovery arm [--ttl 30m]
  netductor recovery disarm
  netductor recovery status
  netductor recovery knock-show
  netductor recovery knock-regen

Default: recovery HTTP is OFF. Arm via CLI or host-specific port-knock sequence.
`)
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "arm":
		ttl := 30 * time.Minute
		for i := 1; i < len(args); i++ {
			if args[i] == "--ttl" && i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				ttl = d
				i++
			}
		}
		if err := secondary.ArmRecovery(ttl); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		armed, until := secondary.RecoveryStatus()
		fmt.Printf("armed=%v until=%s\n", armed, until.Format(time.RFC3339))
	case "disarm", "off":
		secondary.DisarmRecovery()
		fmt.Println("disarmed")
	case "status":
		armed, until := secondary.RecoveryStatus()
		if !armed {
			fmt.Println("armed=false")
			return
		}
		fmt.Printf("armed=true until=%s\n", until.Format(time.RFC3339))
	case "knock-show", "knock":
		ports, err := secondary.KnockPorts()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		parts := make([]string, len(ports))
		for i, p := range ports {
			parts[i] = strconv.Itoa(p)
		}
		fmt.Println(strings.Join(parts, " "))
		fmt.Fprintf(os.Stderr, "# order matters; window 8s between steps; from same IP\n")
		fmt.Fprintf(os.Stderr, "# example: for p in %s; do nc -z SECONDARY $p; sleep 0.3; done\n", strings.Join(parts, " "))
	case "knock-regen":
		ports, err := secondary.RegenerateKnockPorts()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		parts := make([]string, len(ports))
		for i, p := range ports {
			parts[i] = strconv.Itoa(p)
		}
		fmt.Println(strings.Join(parts, " "))
		fmt.Fprintln(os.Stderr, "# new sequence written; restart agent/knock watchers to bind new ports")
	default:
		fmt.Fprintln(os.Stderr, "unknown recovery subcommand")
		os.Exit(2)
	}
}
