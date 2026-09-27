package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/install"
)

func runAPIPublic(args []string) {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, `usage:
  netductor api-public arm [--ttl 15m]
  netductor api-public disarm
  netductor api-public status

Temporarily open mTLS :8789 from the public Internet (UFW).
Auto-disarm via systemd timer. Prefer TG button or this CLI on primary.
Default TTL 15m, min 1m, max 2h.
`)
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "arm":
		ttl := 15 * time.Minute
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
		if err := install.ArmAPIPublic(ttl); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		armed, until := install.APIPublicStatus()
		fmt.Printf("armed=%v until=%s\n", armed, until.UTC().Format(time.RFC3339))
	case "disarm", "off":
		if err := install.DisarmAPIPublic(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("armed=false")
	case "status":
		armed, until := install.APIPublicStatus()
		if !armed {
			fmt.Println("armed=false")
			return
		}
		fmt.Printf("armed=true until=%s\n", until.UTC().Format(time.RFC3339))
	default:
		fmt.Fprintln(os.Stderr, "unknown api-public subcommand")
		os.Exit(2)
	}
}
