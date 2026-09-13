package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/sites"
)

func runSites(args []string) {
	if len(args) < 1 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		list, err := sites.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("no sites")
			return
		}
		for _, s := range list {
			fmt.Printf("%s\t%s\trpi=%s\tmt=%s\n", s.ID, s.Name, s.RPiID, s.MikroTikID)
		}
	case "add", "upsert":
		id, name, rpi, mt := "", "", "", ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--id":
				if i+1 < len(args) {
					i++
					id = args[i]
				}
			case "--name":
				if i+1 < len(args) {
					i++
					name = args[i]
				}
			case "--rpi":
				if i+1 < len(args) {
					i++
					rpi = args[i]
				}
			case "--mt":
				if i+1 < len(args) {
					i++
					mt = args[i]
				}
			}
		}
		if id == "" {
			fmt.Fprintln(os.Stderr, "required: --id")
			os.Exit(2)
		}
		if name == "" {
			name = id
		}
		s, err := sites.Upsert(sites.Site{ID: id, Name: name, RPiID: rpi, MikroTikID: mt})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("saved %s rpi=%s mt=%s\n", s.ID, s.RPiID, s.MikroTikID)
	case "bootstrap":
		id, name, rpiLAN := "home", "Home", "192.168.88.2"
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--id":
				if i+1 < len(args) {
					i++
					id = args[i]
				}
			case "--name":
				if i+1 < len(args) {
					i++
					name = args[i]
				}
			case "--rpi-lan":
				if i+1 < len(args) {
					i++
					rpiLAN = args[i]
				}
			}
		}
		_, _ = sites.Upsert(sites.Site{ID: id, Name: name})
		rsc, _ := sites.RSCForSite(id)
		fmt.Println("=== SITE BOOTSTRAP ===")
		fmt.Printf("site_id=%s name=%s\n\n", id, name)
		fmt.Println("1) RPi OpenWrt (same agent stack as Cudy):")
		fmt.Println("   flash OpenWrt; LAN toward MikroTik (example " + rpiLAN + "/24)")
		fmt.Println("   install netductor-agent for your arch from GitHub releases")
		fmt.Println("   enroll → approve in Admin/TG → note device_id")
		fmt.Println("   netductor sites add --id " + id + " --name \"" + name + "\" --rpi <device_id>")
		fmt.Println()
		fmt.Println("2) MikroTik (routing only):")
		fmt.Println("   policy route / default via " + rpiLAN)
		fmt.Println("   netductor sites rsc " + id)
		fmt.Println("   or POST /api/sites/push-rsc {site_id,host,user,password}")
		fmt.Println()
		fmt.Println("3) Verify: LAN client → MT → RPi → relay")
		fmt.Println()
		fmt.Println("=== RSC ===")
		fmt.Print(rsc)
		fmt.Printf("\n# Suggested:\n/ip route add dst-address=0.0.0.0/0 gateway=%s routing-table=via-rpi\n", rpiLAN)
	case "rsc":
		id := ""
		if len(args) > 1 {
			id = args[1]
		}
		if id == "" {
			list, _ := sites.List()
			if len(list) > 0 {
				id = list[0].ID
			}
		}
		rsc, err := sites.RSCForSite(id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(rsc)
	case "push":
		id, host, user, pass, rpiLAN := "", "", "admin", "", "192.168.88.2"
		port := 22
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--id":
				if i+1 < len(args) {
					i++
					id = args[i]
				}
			case "--host":
				if i+1 < len(args) {
					i++
					host = args[i]
				}
			case "--user":
				if i+1 < len(args) {
					i++
					user = args[i]
				}
			case "--password":
				if i+1 < len(args) {
					i++
					pass = args[i]
				}
			case "--port":
				if i+1 < len(args) {
					i++
					fmt.Sscanf(args[i], "%d", &port)
				}
			case "--rpi-lan":
				if i+1 < len(args) {
					i++
					rpiLAN = args[i]
				}
			}
		}
		if id == "" || host == "" || pass == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor sites push --id home --host 192.168.88.1 --user admin --password '...' [--rpi-lan 192.168.88.2] [--port 22]")
			fmt.Fprintln(os.Stderr, "NOTE: SSH must reach the MikroTik from THIS machine (home LAN or VPN into LAN). Core VPS usually cannot see private 192.168.x.x.")
			os.Exit(2)
		}
		_, _ = sites.Upsert(sites.Site{ID: id, Name: id, MikroTikID: "mt-" + id})
		rsc, err := sites.RSCForSiteWithGateway(id, rpiLAN)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := mikrotik.PushRSC(host, user, pass, nil, rsc, port); err != nil {
			fmt.Fprintln(os.Stderr, "push failed:", err)
			os.Exit(1)
		}
		fmt.Println("ok: RSC pushed to", host)
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor sites list|add|bootstrap|rsc|push")
		os.Exit(2)
	}
}

var _ = strings.TrimSpace
