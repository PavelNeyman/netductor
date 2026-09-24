package main

import (
	"fmt"
	"os"
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

Recovery HTTP :8790 is OFF until arm. Run on secondary over SSH, then recover from new primary.
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
	default:
		fmt.Fprintln(os.Stderr, "unknown recovery subcommand (knock removed; use arm over SSH)")
		os.Exit(2)
	}
}
