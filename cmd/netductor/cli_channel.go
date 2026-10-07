package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/channels"
)

func runChannel(args []string) {
	jsonOut := false
	sub := "status"
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
			continue
		}
		if a != "" && a[0] != '-' {
			sub = a
		}
	}
	switch sub {
	case "status", "health":
		r := channels.Collect()
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(r)
			return
		}
		fmt.Print(channels.FormatText(r))
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor channel status [--json]")
		os.Exit(2)
	}
}
