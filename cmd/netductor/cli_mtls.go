package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/cli18n"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func runMTLS(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Print(cli18n.T("mtls.help"))
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
		fmt.Println(cli18n.T("mtls.ready"), mtls.Dir())
		fmt.Println("  server:", mtls.ServerReady(), " client:", mtls.ClientReady())
	case "issue-client":
		if len(args) < 2 || args[1] == "" {
			fmt.Fprintln(os.Stderr, cli18n.T("mtls.usage.issue"))
			os.Exit(2)
		}
		_, _, _, err := mtls.EnsureClientFor(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(cli18n.T("mtls.client_material"), mtls.ClientDir(args[1]))
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
			fmt.Println(cli18n.T("mtls.clients_none"))
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
		if pend, err := mtls.PendingRotates(); err == nil && len(pend) > 0 {
			fmt.Println(cli18n.T("mtls.pending_header"))
			for _, p := range pend {
				fmt.Printf("  %s old=%s new=%s expires=%d\n", p.NodeID, p.OldSerial, p.NewSerial, p.ExpiresAt)
			}
		}
	case "revoke":
		if len(args) < 2 || args[1] == "" {
			fmt.Fprintln(os.Stderr, cli18n.T("mtls.usage.revoke"))
			os.Exit(2)
		}
		id := args[1]
		if _, err := mtls.ReadClientInfo(id); err == nil {
			ser, err := mtls.RevokeNodeCert(id, "cli-revoke")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(cli18n.T("mtls.revoked_node"), id, ser)
			return
		}
		if err := mtls.RevokeSerial(id, "", "cli-revoke"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(cli18n.T("mtls.revoked_serial"), id)
	case "rotate":
		if len(args) < 2 || args[1] == "" {
			fmt.Fprintln(os.Stderr, cli18n.T("mtls.usage.rotate"))
			os.Exit(2)
		}
		node := args[1]
		info, _, _, _, err := mtls.RotateClientForPush(node, "cli-rotate")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(cli18n.T("mtls.rotated"), info.NodeID, info.Serial, info.DaysLeft)
		fmt.Println(cli18n.T("mtls.material"), mtls.ClientDir(node))
		// auto-push: enqueue agent command
		if id := edge.EnqueueCmd(node, "mtls_refresh", ""); id != "" {
			fmt.Println(cli18n.T("mtls.enqueued_edge"), id)
		} else if err := secondary.EnqueueCmd(node, "mtls_refresh"); err == nil {
			fmt.Println(cli18n.T("mtls.enqueued_secondary"))
		} else {
			fmt.Println(cli18n.T("mtls.push_manual"))
		}
		fmt.Println(cli18n.T("mtls.grace_note"), mtls.GraceHours())
	case "revoked":
		list, err := mtls.ListRevoked()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println(cli18n.T("mtls.empty"))
			return
		}
		for _, e := range list {
			fmt.Printf("%s node=%s reason=%s at=%d\n", e.Serial, e.NodeID, e.Reason, e.RevokedAt)
		}
	case "rollover":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: mtls rollover start|status|issue-all|finish|abort")
			os.Exit(2)
		}
		switch args[1] {
		case "start":
			if err := mtls.CARolloverStart(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(cli18n.T("mtls.rollover_started"))
		case "status":
			st := mtls.CARolloverStatus()
			for k, v := range st {
				fmt.Printf("%s=%v\n", k, v)
			}
		case "issue":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: mtls rollover issue <node-id>")
				os.Exit(2)
			}
			_, _, _, err := mtls.IssueClientFromNewCA(args[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			_ = edge.EnqueueCmd(args[2], "mtls_refresh", "")
			_ = secondary.EnqueueCmd(args[2], "mtls_refresh")
			fmt.Println(cli18n.T("mtls.rollover_issued"), args[2])
		case "finish":
			if err := mtls.CARolloverFinish(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(cli18n.T("mtls.rollover_done"))
		case "abort":
			if err := mtls.CARolloverAbort(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(cli18n.T("mtls.rollover_abort"))
		default:
			fmt.Fprintln(os.Stderr, "unknown rollover subcommand")
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, cli18n.T("mtls.unknown"))
		os.Exit(2)
	}
}
