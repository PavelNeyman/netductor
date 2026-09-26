package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/dnsblock"
)

func runDNS(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor dns list|on <id>|off <id>|reload")
		os.Exit(2)
	}
	switch args[0] {
	case "list", "ls":
		cat := dnsblock.Catalog()
		if len(args) > 1 && (args[1] == "-j" || args[1] == "--json") {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(cat)
			return
		}
		for _, e := range cat {
			st := "off"
			if e.Enabled {
				st = "on"
			}
			fmt.Printf("%s\t%s\n", e.ID, st)
		}
	case "on", "enable":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor dns on <id>")
			os.Exit(2)
		}
		if err := dnsblock.SetEnabled(args[1], true); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok", args[1], "on")
	case "off", "disable":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor dns off <id>")
			os.Exit(2)
		}
		if err := dnsblock.SetEnabled(args[1], false); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok", args[1], "off")
	case "reload":
		if err := dnsblock.ReloadBlocky(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok reload")
	default:
		fmt.Fprintln(os.Stderr, "unknown dns subcommand:", args[0])
		fmt.Fprintln(os.Stderr, "usage: netductor dns list|on <id>|off <id>|reload")
		os.Exit(2)
	}
}
