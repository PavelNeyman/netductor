package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/channels"
)

func runChannel(args []string) {
	o := parseCLIOpts(args)
	if o.Help {
		fmt.Fprintln(os.Stderr, "usage: netductor channel status [--json]")
		os.Exit(0)
	}
	sub := "status"
	if len(o.Rest) > 0 {
		sub = o.Rest[0]
	}
	switch sub {
	case "status", "health":
		r := channels.Collect()
		if o.JSON {
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
