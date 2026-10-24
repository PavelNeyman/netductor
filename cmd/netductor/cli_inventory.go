package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/inventory"
)

func runInventory(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor inventory list|add [--id ID] [--name N] --ip IP")
		os.Exit(2)
	}
	path := inventory.DefaultFile
	if v := os.Getenv("NETDUCTOR_INVENTORY"); v != "" {
		path = v
	}
	switch args[0] {
	case "list":
		items, err := inventory.List(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(items)
	case "add":
		id, name, ip := "", "", ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--id":
				if i+1 < len(args) {
					id = args[i+1]
					i++
				}
			case "--name":
				if i+1 < len(args) {
					name = args[i+1]
					i++
				}
			case "--ip":
				if i+1 < len(args) {
					ip = args[i+1]
					i++
				}
			}
		}
		if ip == "" {
			fmt.Fprintln(os.Stderr, "need --ip")
			os.Exit(2)
		}
		s, err := inventory.Register(path, id, name, ip, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(s)
	default:
		fmt.Fprintln(os.Stderr, "unknown inventory command")
		os.Exit(2)
	}
}
