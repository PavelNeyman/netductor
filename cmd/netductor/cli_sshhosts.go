package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func runSSHHosts(args []string) {
	if len(args) < 1 {
		args = []string{"list"}
	}
	kind := "all"
	rest := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--kind" && i+1 < len(args) {
			kind = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	if len(rest) < 1 {
		rest = []string{"list"}
	}
	switch rest[0] {
	case "list":
		if kind == "all" || kind == "mt" || kind == "mikrotik" {
			fmt.Println("=== mikrotik ===")
			for _, e := range mikrotik.ListKnownHosts() {
				fmt.Printf("%s\t%s\n", e.ID, e.KeyPrefix)
			}
		}
		if kind == "all" || kind == "relay" {
			fmt.Println("=== relay ===")
			for _, e := range secondary.ListSSHHosts() {
				fmt.Printf("%s\t%s\n", e.ID, e.KeyPrefix)
			}
		}
	case "forget", "delete", "rm":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor ssh-hosts forget [--kind mt|relay] <id>")
			os.Exit(2)
		}
		id := rest[1]
		var err error
		switch kind {
		case "relay":
			err = secondary.ForgetSSHHost(id)
		case "mt", "mikrotik":
			err = mikrotik.ForgetKnownHost(id)
		default:
			e1 := mikrotik.ForgetKnownHost(id)
			e2 := secondary.ForgetSSHHost(id)
			if e1 != nil && e2 != nil {
				err = fmt.Errorf("mt: %v; relay: %v", e1, e2)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok: forgot", id)
	case "clear":
		switch kind {
		case "relay":
			_ = secondary.ClearSSHHosts()
		case "mt", "mikrotik":
			_ = mikrotik.ClearKnownHosts()
		default:
			_ = mikrotik.ClearKnownHosts()
			_ = secondary.ClearSSHHosts()
		}
		fmt.Println("ok: cleared", kind)
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor ssh-hosts list|forget|clear [--kind mt|relay]")
		os.Exit(2)
	}
}
