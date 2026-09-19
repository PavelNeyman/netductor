package main

import (
	"fmt"
	"time"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/sites"
)

func runEdgeCLI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor edge list|pending|approve|deny|revoke|register|recovery|export|import|set-site|cmd …")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		runEdgeList()
	case "pending":
		for _, d := range edge.ListPending() {
			fmt.Printf("%s\t%s\t%s\t%s\n", d.DeviceID, d.Board, d.WANIP, d.Hostname)
		}
	case "approve":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge approve <device_id>")
			os.Exit(2)
		}
		tok, err := edge.Approve(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "edge.approve", args[1], "")
		fmt.Println("approved", args[1], "token="+tok)
	case "deny":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge deny <device_id>")
			os.Exit(2)
		}
		if err := edge.Deny(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "edge.deny", args[1], "")
		fmt.Println("denied")
	case "revoke":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge revoke <device_id>")
			os.Exit(2)
		}
		if err := edge.Revoke(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "edge.revoke", args[1], "")
		fmt.Println("revoked")
	case "cmd":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge cmd <device_id> <action> [arg]")
			os.Exit(2)
		}
		arg := ""
		if len(args) > 3 {
			arg = strings.Join(args[3:], " ")
		}
		id := edge.EnqueueCmd(args[1], args[2], arg)
		if id == "" {
			fmt.Fprintln(os.Stderr, "enqueue failed (device not approved?)")
			os.Exit(1)
		}
		fmt.Println(id)
	case "register":
		// netductor edge register <device_id> [--site ID] [--note ...]
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge register <device_id> [--site ID] [--note text]")
			os.Exit(2)
		}
		did := args[1]
		site, note := "", ""
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--site":
				if i+1 < len(args) {
					i++
					site = args[i]
				}
			case "--note":
				if i+1 < len(args) {
					i++
					note = args[i]
				}
			}
		}
		if err := edge.RegisterPending(did, site, note); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if site != "" {
			_ = sitesAttach(site, did)
		}
		audit.Log("cli", "edge.register", did, site)
		fmt.Println("registered pending", did)
	case "recovery":
		// issue one-time code for LAN recovery page
		site, note := "", ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--site":
				if i+1 < len(args) {
					i++
					site = args[i]
				}
			case "--note":
				if i+1 < len(args) {
					i++
					note = args[i]
				}
			}
		}
		code, exp, err := edge.IssueRecoveryCode(site, note, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(code)
		fmt.Fprintln(os.Stderr, "expires", exp.Format(time.RFC3339), "site="+site)
		fmt.Fprintln(os.Stderr, "On router LAN open http://<router-ip>:7879/netductor-recovery")
	case "export":
		b, err := edge.ExportDevices()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		out := "/tmp/nd-edge-devices.json"
		for i := 1; i < len(args); i++ {
			if args[i] == "-o" && i+1 < len(args) {
				out = args[i+1]
			}
		}
		if err := os.WriteFile(out, append(b, 10), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(out)
	case "import":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge import <file> [--replace]")
			os.Exit(2)
		}
		raw, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		replace := false
		for _, a := range args[2:] {
			if a == "--replace" {
				replace = true
			}
		}
		n, err := edge.ImportDevices(raw, replace)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("imported", n)
	case "set-site":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge set-site <device_id> <site_id>")
			os.Exit(2)
		}
		if err := edge.SetSiteID(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = sitesAttach(args[2], args[1])
		fmt.Println("ok")
	case "provision":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge provision user@host --id DEVICE [--server URL] [--key KEY] [--agent BIN]")
			os.Exit(2)
		}
		opts := edge.ProvisionOpts{SSHTarget: args[1], ServerURL: envOr("NETDUCTOR_PUBLIC_URL", "")}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--id":
				if i+1 < len(args) {
					i++
					opts.DeviceID = args[i]
				}
			case "--server":
				if i+1 < len(args) {
					i++
					opts.ServerURL = args[i]
				}
			case "--key":
				if i+1 < len(args) {
					i++
					opts.SSHKey = args[i]
				}
			case "--agent":
				if i+1 < len(args) {
					i++
					opts.AgentBin = args[i]
				}
			}
		}
		if opts.DeviceID == "" || opts.ServerURL == "" {
			fmt.Fprintln(os.Stderr, "--id and --server (or NETDUCTOR_PUBLIC_URL) required")
			os.Exit(2)
		}
		if err := edge.Provision(opts); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("provisioned", opts.DeviceID, "→ pending enroll")
	case "templates":
		edge.EnsureDefaultTemplate()
		for _, tmpl := range edge.ListTemplates() {
			fmt.Println(tmpl["id"])
		}
	case "bind-template":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge bind-template <device_id> <template_id>")
			os.Exit(2)
		}
		if err := edge.BindTemplate(args[1], args[2], nil); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("bound")
	case "mikrotik-rsc":
		name, relayIP, coreURL := "mt-device", "", "http://127.0.0.1:8787"
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--name":
				if i+1 < len(args) {
					name = args[i+1]
					i++
				}
			case "--secondary":
				if i+1 < len(args) {
					relayIP = args[i+1]
					i++
				}
			case "--core":
				if i+1 < len(args) {
					coreURL = args[i+1]
					i++
				}
			}
		}
		if relayIP == "" {
			for _, d := range secondary.List() {
				if d.PublicIP != "" {
					relayIP = d.PublicIP
					break
				}
			}
		}
		fmt.Print(mikrotik.ClientRSC(name, relayIP, coreURL, "netductor"))
	default:
		fmt.Fprintln(os.Stderr, "unknown edge subcommand")
		os.Exit(2)
	}
}

func runEdgeList() {
	devs := edge.ListDevices()
	for _, d := range devs {
		st := "offline"
		if d.Healthy {
			st = "online"
		}
		fmt.Printf("%s\t%s\t%s\t%s\t%s\tip=%s\tlast=%d\n",
			d.DeviceID, d.Status, st, d.Board, d.Hostname, d.WANIP, d.LastSeen)
	}
}


func sitesAttach(siteID, deviceID string) error {
	return sites.AttachEdge(siteID, deviceID)
}
