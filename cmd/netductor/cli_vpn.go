package main

import (
	"github.com/PavelNeyman/netductor/internal/audit"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"os"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/relay"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runVPN(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: netductor vpn add|list|link|note|disable|enable|revoke|apply|set-sni|session ...")
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
	case "add":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "name required")
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
			fmt.Fprintln(os.Stderr, "usage: netductor vpn client-config <name>")
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
			fmt.Fprintln(os.Stderr, "user not found")
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
		kind := "sub"
		if len(rest) > 1 {
			kind = rest[1]
		}
		name := rest[0]
		var s string
		var ok bool
		switch kind {
		case "core":
			s, ok = vpn.ReadClient(name, "link-vless.txt", "link.txt")
		case "hy2":
			s, ok = vpn.ReadClient(name, "link-hy2.txt")
		case "vless":
			// fall through to preferred (relay-first)
			fallthrough
		default:
			// preferred: relay VLESS when online, else core subscription
			r, err := vpn.ListNative()
			if err == nil {
				for _, u := range r {
					if u.Name == name {
						vless := vpn.VLESSLink(name, u.UUID)
						via := "core"
						for _, d := range relay.List() {
							if relay.Online(d, 2*time.Minute) && d.PublicIP != "" && d.PBK != "" {
								sni := d.SNI
								if sni == "" {
									sni = "ya.ru"
								}
								vless = vpn.ClientLinkForRelay(name, u.UUID, d.PublicIP, d.PBK, d.SID, sni)
								via = "relay:" + d.ID
								break
							}
						}
						fmt.Println(vless)
						if via != "core" {
							fmt.Fprintln(os.Stderr, "via", via)
						}
						if kind != "vless" {
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
				fmt.Fprintln(os.Stderr, "not found")
				os.Exit(1)
			}
			fmt.Println(s)
			return
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "not found")
			os.Exit(1)
		}
		fmt.Println(s)
	case "set-sni":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "usage: netductor vpn set-sni <hostname|preset>")
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
		relay.BumpConfigVer()
		fmt.Println("sni set to", sniName)
	case "sni":
		fmt.Println("active", vpn.ActiveSNI())
		for _, p := range vpn.ListSNIPresets() {
			mark := ""
			if p.SNI == vpn.ActiveSNI() {
				mark = " *"
			}
			fmt.Printf("%s\t%s\t%s%s\n", p.Name, p.SNI, p.Note, mark)
		}
	case "apply":
		relay.BumpConfigVer()
		if err := vpn.ApplyConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("config applied")
	case "session":
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
		fmt.Fprintf(os.Stderr, "expires_unix=%d hours=%d\n", exp, hours)
	case "edge-list":
		runEdgeList()
	case "edge-cmd":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor vpn edge-cmd <device_id> <action> [arg]")
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

