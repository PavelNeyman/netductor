package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runServices(args []string) {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list", "ls":
		c, err := policy.EnsureCatalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		b, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(b))
	case "ensure":
		c, err := policy.EnsureCatalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("catalog %d services at %s\n", len(c.Services), policy.CatalogPath())
	case "set":
		// services set <id> --title T --kind internal|egress
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: services set <id> [--title T] [--kind internal|egress] [--desc D]")
			os.Exit(2)
		}
		s := policy.Service{ID: args[1], Kind: policy.KindInternal}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--title":
				if i+1 < len(args) {
					s.Title, i = args[i+1], i+1
				}
			case "--kind":
				if i+1 < len(args) {
					s.Kind, i = args[i+1], i+1
				}
			case "--desc", "--description":
				if i+1 < len(args) {
					s.Description, i = args[i+1], i+1
				}
			}
		}
		if s.Title == "" {
			s.Title = s.ID
		}
		c, err := policy.EnsureCatalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := c.Upsert(s); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := policy.SaveCatalog(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		audit.Log("cli", "services.set", s.ID, s.Kind)
		fmt.Println("ok", s.ID)
	case "disable":
		if len(args) < 2 {
			os.Exit(2)
		}
		c, err := policy.EnsureCatalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := c.SoftDisable(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = policy.SaveCatalog(c)
		fmt.Println("disabled", args[1])
	default:
		fmt.Fprintln(os.Stderr, "usage: services list|ensure|set|disable")
		os.Exit(2)
	}
}

func runPolicyCLI(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: policy apply | policy user get|set <name> | policy edge get|set <id>")
		os.Exit(2)
	}
	switch args[0] {
	case "apply":
		st, err := policy.ApplyRoutes()
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(b))
		if err != nil {
			os.Exit(1)
		}
	case "user":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: policy user get <name> | policy user set <name> [--internet true|false] [--all] [--service id]*")
			os.Exit(2)
		}
		sub, name := args[1], args[2]
		if sub == "get" {
			p, err := vpn.GetUserPolicy(name)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			b, _ := json.MarshalIndent(p, "", "  ")
			fmt.Println(string(b))
			return
		}
		if sub == "set" {
			p, err := vpn.GetUserPolicy(name)
			if err != nil {
				// start from default if we want set-on-create? require existing user
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			p = parsePolicyFlags(args[3:], p)
			if err := vpn.SetUserPolicy(name, p, "cli"); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			out, _ := vpn.GetUserPolicy(name)
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
			return
		}
		os.Exit(2)
	case "edge":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: policy edge get <id> | policy edge set <id> ...")
			os.Exit(2)
		}
		sub, id := args[1], args[2]
		if sub == "get" {
			p, err := edge.GetDevicePolicy(id)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			b, _ := json.MarshalIndent(p, "", "  ")
			fmt.Println(string(b))
			return
		}
		if sub == "set" {
			p, err := edge.GetDevicePolicy(id)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			p = parsePolicyFlags(args[3:], p)
			if err := edge.SetDevicePolicy(id, p, "cli"); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			out, _ := edge.GetDevicePolicy(id)
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
			return
		}
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "unknown policy subcommand")
		os.Exit(2)
	}
}

func parsePolicyFlags(args []string, p policy.AccessPolicy) policy.AccessPolicy {
	servicesSet := false
	var services []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--internet":
			if i+1 < len(args) {
				p.AllowInternet = strings.EqualFold(args[i+1], "true") || args[i+1] == "1" || args[i+1] == "yes"
				i++
			}
		case "--all":
			p.ServicesMode = "all"
		case "--list":
			p.ServicesMode = "list"
		case "--service", "--svc":
			if i+1 < len(args) {
				services = append(services, args[i+1])
				servicesSet = true
				i++
			}
		}
	}
	if servicesSet {
		p.Services = services
		if p.ServicesMode == "" {
			p.ServicesMode = "list"
		}
	}
	p.Normalize()
	return p
}
