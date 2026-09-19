package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/nodes"
)

func runFleet(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage: netductor fleet <cmd>
  status | bootstrap
  set-primary <id> | set-secondary <id>
  provision-secondary --host --password [--user root] [--port 22] [--sni SNI]
  disable-legacy   stop old sync/bot-failover units on this host

removed (secondary is VPN entry only):
  sync | sync-timer | apply-lampac (use: netductor install lampac on primary)
  bot-failover

VPN users → secondary: automatic on vpn add (config_ver); force: netductor secondary sync`)
		os.Exit(2)
	}
	switch args[0] {
	case "status":
		fmt.Print(fleet.StatusSummary())
	case "set-primary":
		if len(args) < 2 {
			os.Exit(2)
		}
		if err := fleet.SetPrimary(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("primary", args[1])
	case "set-secondary":
		if len(args) < 2 {
			os.Exit(2)
		}
		if err := fleet.SetSecondary(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("secondary", args[1])
	case "sync", "sync-timer", "apply-lampac", "bot-failover":
		fmt.Fprintln(os.Stderr, "removed:", args[0], "— secondary is VPN entry only (see docs/PLAN-SECONDARY-VPN-ONLY.md)")
		if args[0] == "apply-lampac" {
			fmt.Fprintln(os.Stderr, "hint: netductor install lampac   # on primary")
		}
		if args[0] == "sync" || args[0] == "sync-timer" {
			_ = fleet.InstallSyncTimer()
		}
		os.Exit(0)
	case "disable-legacy":
		fleet.DisableLegacyFleetUnits()
		fmt.Println("disabled legacy fleet-sync / bot-failover units (if present)")
	case "provision-secondary":
		host, user, pass, sni := "", "root", "", ""
		port := 22
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--host" && i+1 < len(args):
				i++; host = args[i]
			case a == "--user" && i+1 < len(args):
				i++; user = args[i]
			case a == "--password" && i+1 < len(args):
				i++; pass = args[i]
			case a == "--port" && i+1 < len(args):
				i++; fmt.Sscanf(args[i], "%d", &port)
			case a == "--sni" && i+1 < len(args):
				i++; sni = args[i]
			case a == "--no-lampac", a == "--no-bot-standby":
				// ignored; always VPN-entry only
			}
		}
		if host == "" || pass == "" {
			fmt.Fprintln(os.Stderr, "required: --host and --password")
			os.Exit(2)
		}
		if err := fleet.ProvisionSecondary(fleet.ProvisionSecondaryOpts{
			Host: host, User: user, Password: pass, Port: port, SNI: sni,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "bootstrap":
		if err := fleetBootstrap(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(fleet.StatusSummary())
	default:
		fmt.Fprintln(os.Stderr, "unknown fleet subcommand")
		os.Exit(2)
	}
}

func fleetBootstrap() error {
	list, err := nodes.List()
	if err != nil {
		return err
	}
	var coreID, relayID string
	for _, n := range list {
		if n.Role == "core" || (n.Role == "" && strings.Contains(n.Hostname, "core")) {
			if coreID == "" {
				coreID = n.ID
			}
		}
		if n.Role == "secondary" && n.Status == "online" {
			relayID = n.ID
		}
	}
	if coreID == "" {
		for _, n := range list {
			if n.Kind == "vps" && n.Role != "secondary" {
				coreID = n.ID
				break
			}
		}
	}
	if coreID == "" {
		return fmt.Errorf("no core node in registry — install/recover first")
	}
	if err := fleet.SetPrimary(coreID); err != nil {
		return err
	}
	_, _ = nodes.SetDesiredHostname(coreID, "nd-primary")
	if relayID != "" {
		if err := fleet.SetSecondary(relayID); err != nil {
			return err
		}
		_, _ = nodes.SetDesiredHostname(relayID, "nd-secondary")
	}
	_ = fleet.SetServiceNode("bot", coreID)
	// lampac stays on primary when installed — do not pin to secondary
	_ = fleet.SetServiceNode("lampac", coreID)
	return nil
}
