package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/install"
)

func runAgentAllowlist(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor agent-allowlist list
  netductor agent-allowlist add <ip>
  netductor agent-allowlist apply   — re-sync ufw from file`)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		ips := install.LoadAgentAllowlist()
		if len(ips) == 0 {
			fmt.Println("(empty — :8789 closed until an agent IP is added)")
			return
		}
		for _, ip := range ips {
			fmt.Println(ip)
		}
	case "add":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor agent-allowlist add <ip>")
			os.Exit(2)
		}
		if err := install.AllowAgentMTLSFromIP(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("allowed", strings.TrimSpace(args[1]))
	case "apply":
		if err := install.ApplyAgentAllowlistFirewall(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ufw synced")
	default:
		fmt.Fprintln(os.Stderr, "unknown agent-allowlist subcommand")
		os.Exit(2)
	}
}
