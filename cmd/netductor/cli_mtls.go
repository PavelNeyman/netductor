package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func runMTLS(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Print(`netductor mtls ensure                 — generate CA/server/client certs for agent plane
netductor mtls issue-client <id>     — per-node client cert under secrets/mtls/clients/<id>
netductor mtls list                  — plane + per-node certs (serial, expiry)
netductor mtls revoke <serial|node>  — revoke by serial hex or node id
netductor mtls rotate <node-id>      — re-issue client cert and revoke previous serial
netductor mtls revoked               — list revoke entries

After rotate: re-run edge provision / secondary provision (or scp clients/<id>/) so the device gets new material.
`)
		return
	}
	switch args[0] {
	case "ensure":
		pub := ""
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_ip")); err == nil {
			pub = strings.TrimSpace(string(b))
		}
		if err := mtls.EnsureAll(pub); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("mtls ready:", mtls.Dir())
		fmt.Println("  server:", mtls.ServerReady(), " client:", mtls.ClientReady())
	case "issue-client":
		if len(args) < 2 || args[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor mtls issue-client <node-id>")
			os.Exit(2)
		}
		_, _, _, err := mtls.EnsureClientFor(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("client material:", mtls.ClientDir(args[1]))
	case "list":
		for _, pc := range mtls.ListPlaneCerts() {
			fmt.Printf("plane %-14s serial=%s days_left=%d not_after=%s\n",
				pc.Name, pc.Serial, pc.DaysLeft, pc.NotAfter.UTC().Format("2006-01-02"))
		}
		clients, err := mtls.ListClientCerts()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(clients) == 0 {
			fmt.Println("clients: (none)")
			return
		}
		for _, c := range clients {
			flag := ""
			if c.Revoked {
				flag = " REVOKED"
			}
			fmt.Printf("client %-16s serial=%s days_left=%d not_after=%s%s\n",
				c.NodeID, c.Serial, c.DaysLeft, c.NotAfter.UTC().Format("2006-01-02"), flag)
		}
	case "revoke":
		if len(args) < 2 || args[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor mtls revoke <serial-hex|node-id>")
			os.Exit(2)
		}
		id := args[1]
		// prefer node if client dir exists
		if _, err := mtls.ReadClientInfo(id); err == nil {
			ser, err := mtls.RevokeNodeCert(id, "cli-revoke")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("revoked node", id, "serial", ser)
			return
		}
		if err := mtls.RevokeSerial(id, "", "cli-revoke"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("revoked serial", id)
	case "rotate":
		if len(args) < 2 || args[1] == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor mtls rotate <node-id>")
			os.Exit(2)
		}
		info, err := mtls.RotateClientFor(args[1], "cli-rotate")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("rotated", info.NodeID, "new_serial", info.Serial, "days_left", info.DaysLeft)
		fmt.Println("material:", mtls.ClientDir(args[1]))
		fmt.Println("re-provision or copy certs to device; old serial is revoked")
	case "revoked":
		list, err := mtls.ListRevoked()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("(empty)")
			return
		}
		for _, e := range list {
			fmt.Printf("%s node=%s reason=%s at=%d\n", e.Serial, e.NodeID, e.Reason, e.RevokedAt)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown mtls subcommand")
		os.Exit(2)
	}
}
