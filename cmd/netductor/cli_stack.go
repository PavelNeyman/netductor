package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/stack"
)

func runStack(args []string) {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" {
		fmt.Print(`netductor stack status
netductor stack apply [vX.Y.Z]
netductor stack rollback
netductor stack watchdog
netductor stack watchdog-install
`)
		return
	}
	switch args[0] {
	case "status":
		st := stack.Collect()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(st)
	case "apply":
		tag := ""
		if len(args) > 1 {
			tag = args[1]
		}
		if err := stack.Apply(tag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "rollback":
		if err := stack.Rollback(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "watchdog":
		stack.WatchdogOnce()
		st := stack.Collect()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(st)
	case "watchdog-install":
		if err := stack.InstallWatchdogTimer(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("stack watchdog timer enabled")
	default:
		fmt.Fprintln(os.Stderr, "unknown stack subcommand:", args[0])
		os.Exit(2)
	}
}
