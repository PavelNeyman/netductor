package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/cleanup"
)

func runCleanup(args []string) {
	apply := false
	asJSON := false
	for _, a := range args {
		switch a {
		case "--apply", "-y":
			apply = true
		case "--json":
			asJSON = true
		case "--help", "-h":
			fmt.Print(`netductor cleanup [--apply|-y] [--json]

Remove upgrade leftovers (stack attempt, /tmp/nd-sb-*, config.json.tmp, legacy test WG/units).
Default is dry-run. Does NOT delete stack/prev, backups, baseline, or live sing-box config.

Legacy alias: netductor cleanup-legacy [--apply]
`)
			return
		}
	}
	r := cleanup.Run(apply)
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		return
	}
	fmt.Print(cleanup.FormatText(r))
	if !apply && len(r.Items) > 0 {
		os.Exit(0)
	}
}

func runCleanupLegacy(args []string) {
	// compat wrapper
	runCleanup(args)
}
