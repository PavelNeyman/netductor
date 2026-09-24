package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/domain"
)

func runDomain(args []string) {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, `usage:
  netductor domain show
  netductor domain set --base netductor.example.com [--http] [--enable-redirect] [--le --email you@x]
  netductor domain set --primary HOST --vpn HOST --redirect URL [--enable-redirect]

Preset from --base:
  primary.<base>  secondary not written (use --vpn)
  vpn.<base>      as VPN entry host
  https://i.<base> as REDIRECT_BASE (--http → http://)
`)
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "show", "status":
		domain.Show()
	case "set":
		var c domain.Config
		for i := 1; i < len(args); i++ {
			a := args[i]
			next := func() string {
				if i+1 < len(args) {
					i++
					return args[i]
				}
				return ""
			}
			switch a {
			case "--base":
				c.Base = next()
			case "--primary", "--core":
				c.Primary = next()
			case "--vpn", "--secondary":
				c.VPN = next()
			case "--redirect":
				c.RedirectBase = next()
			case "--http":
				c.UseHTTPRedirect = true
			case "--enable-redirect":
				c.EnableRedirect = true
			case "--le", "--letsencrypt":
				c.LE = true
				c.EnableRedirect = true
			case "--email":
				c.LEEmail = next()
			case "--staging":
				c.LEStaging = true
			}
		}
		if err := domain.Apply(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown domain subcommand")
		os.Exit(2)
	}
}
