package main

import (
	"github.com/PavelNeyman/netductor/internal/cli18n"
	"encoding/json"
	"github.com/PavelNeyman/netductor/internal/audit"
	"fmt"
	"path/filepath"
	"strings"
	"os"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/session"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runVPN(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, cli18n.T("vpn.usage"))
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "list":
		users, err := vpn.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, u := range users {
			en := "off"
			if u.Enabled {
				en = "on"
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n", u.Name, en, u.UUID, u.Note, u.Created)
		}
	case "rename":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.usage.rename"))
			os.Exit(2)
		}
		out, err := vpn.Rename(rest[0], rest[1])
		fmt.Println(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "vpn.rename", rest[0], rest[1])
	case "add":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.name_required"))
			os.Exit(2)
		}
		note := ""
		if len(rest) > 1 {
			note = rest[1]
		}
		out, err := vpn.Add(rest[0], note)
		fmt.Println(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "vpn.add", rest[0], note)
	case "note":
		if len(rest) < 1 {
			os.Exit(2)
		}
		note := ""
		if len(rest) > 1 {
			note = rest[1]
		}
		out, err := vpn.Note(rest[0], note)
		fmt.Println(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "disable", "enable", "revoke":
		if len(rest) < 1 {
			os.Exit(2)
		}
		var out string
		var err error
		switch cmd {
		case "disable":
			out, err = vpn.Disable(rest[0])
		case "enable":
			out, err = vpn.Enable(rest[0])
		case "revoke":
			out, err = vpn.Revoke(rest[0])
		}
		fmt.Println(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "client-config":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.usage.client_config"))
			os.Exit(2)
		}
		name := args[1]
		users, _ := vpn.ListNative()
		var uuid string
		for _, u := range users {
			if u.Name == name {
				uuid = u.UUID
				break
			}
		}
		if uuid == "" {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.user_not_found"))
			os.Exit(1)
		}
		if err := vpn.WriteClientConfigs(name, uuid); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(filepath.Join(vpn.Clients(), name))
	case "link":
		if len(rest) < 1 {
			os.Exit(2)
		}
		kind := "vless"
		if len(rest) > 1 {
			kind = rest[1]
		}
		name := rest[0]
		var s string
		var ok bool
		switch kind {
		case "sub", "subscription", "sub64", "b64":
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.sub_removed"))
			os.Exit(2)
		case "core":
			s, ok = vpn.ReadClient(name, "link-vless-core.txt", "link-vless.txt", "link.txt")
		case "hy2":
			s, ok = vpn.ReadClient(name, "link-hy2.txt")
		case "vless":
			// fall through to preferred (secondary-first)
			fallthrough
		default:
			// preferred: secondary VLESS when online, else core
			r, err := vpn.ListNative()
			if err == nil {
				for _, u := range r {
					if u.Name == name {
						vless := vpn.PreferredVLESSLink(name, u.UUID)
						via := "core"
						if e := vpn.ResolveClientEndpoints(name, u.UUID); e.SecondaryHost != "" {
							via = "secondary:" + e.SecondaryHost
						}
						fmt.Println(vless)
						if via != "core" {
							fmt.Fprintln(os.Stderr, cli18n.T("vpn.via", via))
						}
						if kind == "full" || kind == "all" {
							if core, ok2 := vpn.ReadClient(name, "link-vless-core.txt"); ok2 {
								fmt.Println(strings.TrimSpace(core))
							}
							if hy, ok2 := vpn.ReadClient(name, "link-hy2.txt"); ok2 {
								fmt.Println(strings.TrimSpace(hy))
							}
						}
						return
					}
				}
			}
			s, ok = vpn.ReadClient(name, "subscription.txt", "link.txt")
		}
		if kind == "vless" || kind == "hy2" {
			if !ok {
				fmt.Fprintln(os.Stderr, cli18n.T("vpn.not_found"))
				os.Exit(1)
			}
			fmt.Println(s)
			return
		}
		if !ok {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.not_found"))
			os.Exit(1)
		}
		fmt.Println(s)
	case "refresh-links":
		only := ""
		if len(rest) > 0 {
			only = rest[0]
		}
		n, err := vpn.RefreshLinks(only)
		fmt.Printf(cli18n.T("vpn.refreshed")+"\n", n)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "wl-check":
		ip := ""
		if len(rest) > 0 {
			ip = rest[0]
		}
		var r vpn.WLCheckResult
		if ip == "" {
			r = vpn.CheckSelfPublicIP()
		} else {
			r = vpn.CheckWhitelistIP(ip)
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
	case "mismatch":
		st := vpn.CollectMismatch(30)
		fmt.Println(vpn.FormatMismatchText(st))
		fmt.Println(cli18n.T("vpn.secondary_hdr"))
		for _, d := range secondary.List() {
			if d.MismatchTotal == 0 && len(d.MismatchByIP) == 0 {
				continue
			}
			fmt.Printf("%s %s total=%d %v\n", d.ID, d.PublicIP, d.MismatchTotal, d.MismatchByIP)
		}
	case "set-sni":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.usage.set_sni"))
			os.Exit(2)
		}
		sniName := rest[0]
		for _, p := range vpn.ListSNIPresets() {
			if p.Name == sniName {
				sniName = p.SNI
				break
			}
		}
		if err := vpn.SetSNI(sniName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		secondary.BumpConfigVer()
		fmt.Println(cli18n.T("vpn.sni_set", sniName))
	case "sni-import":
		url := ""
		if len(rest) > 0 {
			url = rest[0]
		}
		n, err := vpn.ImportSNIPresetsFromURL(url)
		fmt.Printf(cli18n.T("vpn.presets")+"\n", n)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "sni":
		fmt.Println(cli18n.T("vpn.active", vpn.ActiveSNI()))
		for _, p := range vpn.ListSNIPresets() {
			mark := ""
			if p.SNI == vpn.ActiveSNI() {
				mark = " *"
			}
			fmt.Printf("%s\t%s\t%s%s\n", p.Name, p.SNI, p.Note, mark)
		}
	case "apply":
		dry := false
		for _, a := range rest {
			if a == "--dry-run" || a == "-n" {
				dry = true
			}
		}
		if dry {
			msg, err := vpn.ApplyConfigDryRun()
			fmt.Println(msg)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		secondary.BumpConfigVer()
		if err := vpn.ApplyConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(cli18n.T("vpn.config_applied"))
	case "session":
		if len(rest) > 0 && (rest[0] == "list" || rest[0] == "ls") {
			for _, s := range session.List() {
				fmt.Printf("%s\texp=%d\tlabel=%s\tip=%s\n", s.HashPrefix, s.Exp, s.Label, s.IP)
			}
			return
		}
		if len(rest) > 0 && rest[0] == "revoke-all" {
			_ = session.RevokeAll()
			fmt.Println(cli18n.T("vpn.revoked_all"))
			return
		}
		hours := 72
		if len(rest) > 0 {
			fmt.Sscanf(rest[0], "%d", &hours)
		}
		tok, exp, err := vpn.CreateSession(hours)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(tok)
		fmt.Fprintf(os.Stderr, cli18n.T("vpn.expires")+"\n", exp, hours)
	case "edge-list":
		runEdgeList()
	case "edge-cmd":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, cli18n.T("vpn.usage.edge_cmd"))
			os.Exit(2)
		}
		arg := ""
		if len(rest) > 2 {
			arg = rest[2]
		}
		id := edge.EnqueueCmd(rest[0], rest[1], arg)
		fmt.Println(id)
	case "api-bind":
		mode := "localhost"
		ufw := false
		for _, a := range rest {
			if a == "--ufw" {
				ufw = true
				continue
			}
			mode = a
		}
		if err := vpn.APIBind(mode, ufw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown vpn subcommand"); os.Exit(2)
	}
}

