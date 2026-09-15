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
		fmt.Print(`usage: netductor fleet status|set-primary|set-secondary|set-service|sync|bootstrap

  status                     show policy + nodes
  set-primary <node-id>      control-plane source of truth (prefer abroad)
  set-secondary <node-id>    warm replica / RU data-plane
  set-service lampac|bot <node-id>
  sync [user@host]           replicate lampac data + fleet policy to peer
  bootstrap                  set primary=local core, secondary=first online relay
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
	if relayID != "" {
		if err := fleet.SetSecondary(relayID); err != nil {
			return err
		}
		_ = fleet.SetServiceNode("lampac", relayID)
	}
	_ = fleet.SetServiceNode("bot", coreID)
	return nil
}
