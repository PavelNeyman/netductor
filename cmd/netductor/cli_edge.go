package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/cli18n"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/edgestorage"
	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/sites"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func runEdgeCLI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, cli18n.T("edge.usage"))
		os.Exit(2)
	}
	switch args[0] {

	case "storage":
		runEdgeStorageCLI(args[1:])
	case "list":
		runEdgeList()
	case "bootstrap-token":
		// ensure | show | rotate
		sub := "ensure"
		if len(args) > 1 {
			sub = args[1]
		}
		switch sub {
		case "ensure":
			tok, err := edge.EnsureBootstrapToken()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("edge_bootstrap_token ok len=", len(tok))
		case "show":
			tok := edge.BootstrapToken()
			if tok == "" {
				fmt.Fprintln(os.Stderr, "missing — run: netductor edge bootstrap-token ensure")
				os.Exit(1)
			}
			// Never print full token to stdout by default; only length + suffix.
			suf := tok
			if len(suf) > 8 {
				suf = "…" + suf[len(suf)-8:]
			}
			fmt.Println("present suffix=", suf, "len=", len(tok))
		case "rotate":
			tok, err := edge.RotateBootstrapToken()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("rotated len=", len(tok), "(re-prep offline packs)")
		default:
			fmt.Fprintln(os.Stderr, "usage: netductor edge bootstrap-token ensure|show|rotate")
			os.Exit(2)
		}
	case "reissue-token":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge reissue-token <device_id>")
			os.Exit(2)
		}
		tok, err := edge.ReissueToken(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "edge.reissue-token", args[1], "")
		fmt.Println("reissue ok", args[1], "token_pickup=true token="+tok)
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
		fmt.Println(cli18n.T("edge.approved"), args[1], "token="+tok)
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
		fmt.Println(cli18n.T("edge.denied"))
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
		fmt.Println(cli18n.T("edge.revoked"))
	case "cmd":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge cmd <device_id> <action> [arg]")
			os.Exit(2)
		}
		arg := ""
		if len(args) > 3 {
			arg = strings.Join(args[3:], " ")
		}
		// R8: agent_update with bare tag or empty → fill URL|sha|confirm=yes from device arch + SHA256SUMS
		if args[2] == "agent_update" {
			built, err := buildAgentUpdateArg(args[1], arg)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			arg = built
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
		fmt.Println(cli18n.T("edge.registered"), did)
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
		fmt.Println(cli18n.T("edge.imported"), n)
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
		fmt.Println(cli18n.T("edge.ok"))
	case "provision":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge provision user@host --id DEVICE [--server URL] [--token BOOTSTRAP] [--key KEY] [--agent BIN]")
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
			case "--token":
				if i+1 < len(args) {
					i++
					opts.Token = args[i]
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
		fmt.Println(cli18n.T("edge.provisioned"), opts.DeviceID, "→ pending enroll")
	case "template-get":
		id := "default"
		if len(args) > 1 {
			id = args[1]
		}
		edge.EnsureDefaultTemplate()
		tmpl, err := edge.GetTemplate(id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		b, _ := json.MarshalIndent(tmpl, "", "  ")
		fmt.Println(string(b))
	case "template-set-vpn":
		// netductor edge template-set-vpn [id] key=value ...
		// keys: enabled, mode, dns, dns_mode, fallback, soft_fallback
		id := "default"
		kvs := args[1:]
		if len(args) > 1 && !strings.Contains(args[1], "=") {
			id = args[1]
			kvs = args[2:]
		}
		m := map[string]any{}
		for _, kv := range kvs {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				fmt.Fprintln(os.Stderr, "want key=value, got", kv)
				os.Exit(2)
			}
			m[parts[0]] = parts[1]
		}
		tmpl, err := edge.SetTemplateVPN(id, m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok", id, "vpn=", tmpl["vpn"])
	case "templates":
		edge.EnsureDefaultTemplate()
		for _, tmpl := range edge.ListTemplates() {
			fmt.Println(tmpl["id"])
		}
	case "vpn-ensure":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge vpn-ensure <device_id>")
			os.Exit(2)
		}
		links, err := edge.EnsureVPNClient(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// also merge into device template so day-2 apply_template is consistent
		if _, err := edge.TemplateWithVPN(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "warn template:", err)
		}
		b, _ := json.Marshal(links)
		fmt.Println(string(b))
		audit.Log("cli", "edge.vpn-ensure", args[1], links["user"])
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
		fmt.Printf("%s\t%s\t%s\t%s\t%s\tip=%s\tagent=%s\tlast=%d\n",
			d.DeviceID, d.Status, st, d.Board, d.Hostname, d.WANIP, d.Agent, d.LastSeen)
	}
}

func sitesAttach(siteID, deviceID string) error {
	return sites.AttachEdge(siteID, deviceID)
}

// buildAgentUpdateArg expands a release tag (or empty=latest) into URL|sha256|confirm=yes using device arch.
func buildAgentUpdateArg(deviceID, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	// Already fully specified
	if strings.Contains(arg, "|") && strings.Contains(arg, "confirm=yes") {
		return arg, nil
	}
	d, ok := edge.GetDevice(deviceID)
	if !ok {
		return "", fmt.Errorf("device not found: %s", deviceID)
	}
	arch := strings.TrimSpace(d.Arch)
	if arch == "" && d.Extra != nil {
		if v, ok := d.Extra["arch"].(string); ok {
			arch = strings.TrimSpace(v)
		}
	}
	if arch == "" {
		return "", fmt.Errorf("device %s has no arch yet (wait for heartbeat with arch=)", deviceID)
	}
	tag := arg
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			return "", err
		}
	}
	return ndupdate.AgentUpdateArg(tag, arch)
}

func runEdgeStorageCLI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor edge storage list|backup-now|restore|mute|unmute|offline-pack <device_id> [--format]")
		os.Exit(2)
	}
	switch args[0] {
	case "backup-now":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge storage backup-now <device_id>")
			os.Exit(2)
		}
		id := args[1]
		a := edge.EnqueueCmd(id, "config_backup", "")
		b := edge.EnqueueCmd(id, "storage_backup", "")
		fmt.Printf("config_backup=%s storage_backup=%s\n", a, b)
		audit.Log("cli", "edge.storage.backup", id, "")
	case "restore":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge storage restore <device_id> [--format]")
			os.Exit(2)
		}
		id := args[1]
		format := false
		for _, a := range args[2:] {
			if a == "--format" {
				format = true
			}
		}
		if format {
			_ = edge.EnqueueCmd(id, "storage_init_usb", "format=yes")
		}
		cid := edge.EnqueueCmd(id, "storage_restore_remote", "")
		if cid == "" {
			fmt.Fprintln(os.Stderr, "enqueue failed")
			os.Exit(1)
		}
		fmt.Println(cid)
		audit.Log("cli", "edge.storage.restore", id, fmt.Sprintf("format=%v", format))
	case "mute", "unmute":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge storage mute|unmute <device_id>")
			os.Exit(2)
		}
		mute := args[0] == "mute"
		if err := edge.SetUSBAlertMute(args[1], mute); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok mute=", mute)
	case "offline-pack":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor edge storage offline-pack <device_id> [dest_dir]")
			os.Exit(2)
		}
		dest := ""
		if len(args) > 2 {
			dest = args[2]
		}
		if dest == "" {
			dest = "/var/lib/netductor/edge-offline/" + args[1]
		}
		dir, err := edgestorage.WriteOfflinePack(args[1], dest)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(dir)
		audit.Log("cli", "edge.storage.offline_pack", args[1], dir)
	default:
		fmt.Fprintln(os.Stderr, "unknown storage subcommand")
		os.Exit(2)
	}
}
