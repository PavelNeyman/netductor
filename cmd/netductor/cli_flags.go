package main

import (
	"fmt"
	"os"
)

// CF-18.3 shared flag layer: noun verb [options]
// Global-ish flags accepted anywhere in the tail (compat with mixed order).

type cliOpts struct {
	JSON   bool
	DryRun bool
	Help   bool
	Chain  bool // CF-15: multi-hop stack apply
	// Rest is positional args with known flags stripped.
	Rest []string
}

// parseCLIOpts strips --json, --dry-run, -h/--help from args (any position).
// Unknown dashed tokens stay in Rest so subcommands can reject them.
func parseCLIOpts(args []string) cliOpts {
	var o cliOpts
	for _, a := range args {
		switch a {
		case "--json", "-j":
			o.JSON = true
		case "--dry-run":
			o.DryRun = true
		case "--chain":
			o.Chain = true
		case "-h", "--help":
			o.Help = true
		default:
			o.Rest = append(o.Rest, a)
		}
	}
	return o
}




// refuseIfDryRun prints a one-liner and returns true when --dry-run was set.
// Callers must return without mutating state.
func refuseIfDryRun(o cliOpts, action string) bool {
	if !o.DryRun {
		return false
	}
	fmt.Fprintf(os.Stderr, "dry-run: would %s (not executed)\n", action)
	return true
}
