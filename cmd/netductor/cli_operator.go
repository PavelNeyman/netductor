package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/operator"
)

func runOperator(args []string) {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, `usage:
  netductor operator serve [--bind 127.0.0.1] [--port 7373]

Localhost-only operator UI + API (FleetDeploy). Never bind 0.0.0.0.
`)
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "serve":
		o := operator.ServeOpts{Bind: "127.0.0.1", Port: "7373"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			if a == "--bind" && i+1 < len(args) {
				i++
				o.Bind = args[i]
			}
			if a == "--port" && i+1 < len(args) {
				i++
				o.Port = args[i]
			}
		}
		if err := operator.Serve(o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown operator subcommand")
		os.Exit(2)
	}
}
