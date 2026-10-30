package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

// runSSHHostsOp: operator-side TOFU management (same store as deploy SSH).
// Subcommands: list | show | trust | forget
func runSSHHostsOp(args []string) int {
	if len(args) < 1 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		entries, err := secondary.ListSSHHosts()
		if err != nil {
			fmt.Fprintln(os.Stderr, "list:", err)
			return 1
		}
		if len(entries) == 0 {
			fmt.Println("(empty)")
			return 0
		}
		for _, e := range entries {
			fmt.Printf("%s\t%s\n", e.ID, e.KeyPrefix)
		}
		return 0
	case "show", "probe":
		return runSSHTrust(append([]string{}, appendFlagShow(args[1:])...))
	case "trust":
		return runSSHTrust(args[1:])
	case "forget", "delete", "rm":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor-op ssh-hosts forget <host|host:port|host|port>")
			return 2
		}
		id := strings.TrimSpace(args[1])
		if id == "" {
			fmt.Fprintln(os.Stderr, "refuse: empty host id")
			return 2
		}
		if err := secondary.ForgetSSHHost(id); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok: forgot", id)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor-op ssh-hosts list|show|trust|forget")
		return 2
	}
}

func appendFlagShow(args []string) []string {
	out := make([]string, 0, len(args)+1)
	out = append(out, args...)
	hasShow := false
	for _, a := range out {
		if a == "--show" {
			hasShow = true
		}
	}
	if !hasShow {
		out = append(out, "--show")
	}
	return out
}
