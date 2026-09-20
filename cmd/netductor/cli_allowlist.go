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
  netductor agent-allowlist add <ip>   — record secondary (stable) IP
  netductor agent-allowlist apply      — re-sync ufw

Default firewall: :8789 open, auth = mTLS (edge behind NAT OK).
NETDUCTOR_AGENT_ALLOWLIST_STRICT=1 — allow only listed IPs (secondary-only fleets).`)
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		ips := install.LoadAgentAllowlist()
		mode := "default (mTLS, :8789 open)"
		if install.AgentAllowlistStrict() {
			mode = "STRICT (ufw only listed IPs)"
		}
		fmt.Println("mode:", mode)
		if len(ips) == 0 {
			fmt.Println("(empty inventory)")
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
		fmt.Println("recorded", strings.TrimSpace(args[1]))
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
