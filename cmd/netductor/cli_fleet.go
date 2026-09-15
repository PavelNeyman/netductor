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
		fmt.Print(`usage: netductor fleet <cmd>

  status | bootstrap
  set-primary|set-secondary|set-service
  provision-secondary --host IP --password PASS [--sni SNI] [--no-lampac] [--no-bot-standby]
      Full secondary deploy from primary: VPN join + fleet + sync + lampac + bot standby
  sync [user@host] | sync-timer | apply-lampac
  bot-standby-install [user@primary]
  bot-failover check|promote|demote|timer

Naming: operator-facing primary/secondary. Internal VPN agent may still say role=relay.
`)
		os.Exit(2)
	}
	switch args[0] {
	case "status":
		fmt.Print(fleet.StatusSummary())
	case "set-primary":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor fleet set-primary <node-id>")
			os.Exit(2)
		}
		if err := fleet.SetPrimary(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("primary set", args[1])
	case "set-secondary":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor fleet set-secondary <node-id>")
			os.Exit(2)
		}
		if err := fleet.SetSecondary(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("secondary set", args[1])
	case "set-service":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor fleet set-service lampac|bot <node-id>")
			os.Exit(2)
		}
		if err := fleet.SetServiceNode(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("service", args[1], "→", args[2])
	case "sync":
		peer := ""
		if len(args) > 1 {
			peer = args[1]
		}
		if err := fleet.SyncPaths(peer); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("sync ok")
	case "sync-timer":
		if err := fleet.InstallSyncTimer(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("fleet sync timer enabled (hourly)")
	case "apply-lampac":
		if err := fleet.ApplyLampac(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("lampac apply done")
	case "bot-standby-install":
		peer := ""
		if len(args) > 1 {
			peer = args[1]
		}
		if err := fleet.InstallBotStandbyUnits(peer); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("bot standby units installed")
	case "bot-failover":
		sub := "check"
		if len(args) > 1 {
			sub = args[1]
		}
		switch sub {
		case "check":
			msg, err := fleet.CheckBotFailover()
			fmt.Println(msg)
			if err != nil {
				os.Exit(1)
			}
		case "promote":
			if err := fleet.PromoteStandbyBot(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("standby promoted")
		case "demote":
			if err := fleet.DemoteStandbyBot(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("standby demoted")
		case "timer":
			if err := fleet.InstallBotFailoverTimer(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("bot failover timer enabled (2m)")
		default:
			fmt.Fprintln(os.Stderr, "usage: fleet bot-failover check|promote|demote|timer")
			os.Exit(2)
		}
	case "provision-secondary":
		host, user, pass, sni := "", "root", "", ""
		port := 22
		noLampac, noBot := false, false
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
			case a == "--no-lampac":
				noLampac = true
			case a == "--no-bot-standby":
				noBot = true
			}
		}
		if host == "" || pass == "" {
			fmt.Fprintln(os.Stderr, "required: --host and --password")
			os.Exit(2)
		}
		if err := fleet.ProvisionSecondary(fleet.ProvisionSecondaryOpts{
			Host: host, User: user, Password: pass, Port: port, SNI: sni,
			NoLampac: noLampac, NoBotStandby: noBot,
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
		if n.Role == "relay" && n.Status == "online" {
			relayID = n.ID
		}
	}
	if coreID == "" {
		for _, n := range list {
			if n.Kind == "vps" && n.Role != "relay" {
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
		_ = fleet.SetServiceNode("lampac", relayID)
	}
	_ = fleet.SetServiceNode("bot", coreID)
	return nil
}
