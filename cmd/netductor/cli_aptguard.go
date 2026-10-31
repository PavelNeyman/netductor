package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/aptguard"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func runAptGuard(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor aptguard scan|status|policy [show|init]")
		os.Exit(2)
	}
	state := paths.StateDir()
	switch args[0] {
	case "scan":
		host, _ := os.Hostname()
		if host == "" {
			host = "local"
		}
		rep, prev, err := aptguard.RunLocalScan(state, host, "/")
		if err != nil {
			fmt.Fprintf(os.Stderr, "aptguard scan: %v\n", err)
			os.Exit(1)
		}
		added, removed := aptguard.DiffFindings(prev, rep)
		for _, f := range added {
			if f.Severity == aptguard.SevFail || f.Severity == aptguard.SevWarn {
				key := fmt.Sprintf("aptguard:%s:%s", host, f.Code)
				msg := fmt.Sprintf("APT Guard %s on %s: %s (%s)", f.Severity, host, f.Code, f.Message)
				notify.EnqueueAlert(key, msg)
			}
		}
		for _, f := range removed {
			key := fmt.Sprintf("aptguard:%s:%s:cleared", host, f.Code)
			notify.EnqueueAlert(key, fmt.Sprintf("APT Guard cleared on %s: %s", host, f.Code))
			_ = f
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		if rep.Status == aptguard.SevFail {
			os.Exit(1)
		}
	case "status":
		st, detail, sev := aptguard.DoctorSummary(state, 12*time.Hour)
		fmt.Printf("aptguard %s %s\n", st, detail)
		if sev == aptguard.SevFail {
			os.Exit(1)
		}
	case "policy":
		sub := "show"
		if len(args) > 1 {
			sub = args[1]
		}
		switch sub {
		case "init", "ensure":
			p, err := aptguard.EnsurePolicy(state)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("policy v%d written %s\n", p.Version, aptguard.PolicyPath(state))
		case "show":
			p, err := aptguard.EnsurePolicy(state)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			b, _ := os.ReadFile(aptguard.PolicyPath(state))
			if len(b) > 0 {
				os.Stdout.Write(b)
			} else {
				fmt.Printf("version: %d repos: %d\n", p.Version, len(p.Repos))
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown policy subcommand %q\n", sub)
			os.Exit(2)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown aptguard command %q\n", args[0])
		os.Exit(2)
	}
}

func init() {
	_ = strings.TrimSpace
}
