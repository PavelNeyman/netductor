package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func runSvcPaths(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: netductor svc-paths status|apply|health\n")
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
	default:
		fmt.Fprintln(os.Stderr, "unknown svc-paths subcommand")
		os.Exit(2)
	}
}
