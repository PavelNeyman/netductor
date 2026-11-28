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
  netductor api-public permanent on|off|status
  netductor api-public arm [--ttl 15m]
  netductor api-public disarm
  netductor api-public status

permanent on — durable public mTLS :8789 for travel edge (ISP NAT, changing IPs).
  Security is app-layer: bootstrap enroll → pending only; commands need approve + device token + client cert.
arm/disarm — temporary TTL window (legacy; not needed when permanent on).
`)
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "permanent":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor api-public permanent on|off|status")
			os.Exit(2)
		}
		switch strings.ToLower(args[1]) {
		case "on", "enable", "1":
			if err := install.SetEdgePlanePublic(true); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("edge-plane-public=on (:8789 open; pending DMZ at app layer)")
		case "off", "disable", "0":
			if err := install.SetEdgePlanePublic(false); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("edge-plane-public=off (:8789 restricted to SP/PS + api-allow)")
		case "status":
			fmt.Printf("edge-plane-public=%v armed_ttl=%v\n", install.EdgePlanePublicEnabled(), func() bool {
				a, _, _ := install.APIPublicStatusChecked()
				return a
			}())
		default:
			fmt.Fprintln(os.Stderr, "usage: netductor api-public permanent on|off|status")
			os.Exit(2)
		}
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
		armed, until, err := install.APIPublicStatusChecked()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("armed=%v until=%s\n", armed, until.UTC().Format(time.RFC3339))
	case "disarm", "off":
		if err := install.DisarmAPIPublic(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("armed=false")
	case "status":
		fmt.Printf("edge-plane-public=%v\n", install.EdgePlanePublicEnabled())
		armed, until, err := install.APIPublicStatusChecked()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
