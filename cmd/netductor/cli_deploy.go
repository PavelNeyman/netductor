package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

func runDeploy(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage: netductor deploy primary|secondary|edge [flags]

  primary   — bootstrap Debian VPS from this machine (Mac/PC)
  secondary — provision RU VPN entry via primary SSH
  edge      — install agent on OpenWrt (LAN; token from primary)

See: netductor tui → Setup wizard`)
		os.Exit(2)
	}
	switch args[0] {
	case "primary":
	o := deploy.PrimaryOpts{Version: "0.8.1", SNI: "api.vk.me", User: "root"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--host" && i+1 < len(args):
				i++; o.Host = args[i]
			case a == "--user" && i+1 < len(args):
				i++; o.User = args[i]
			case a == "--password" && i+1 < len(args):
				i++; o.Password = args[i]
			case a == "--key" && i+1 < len(args):
				i++; o.SSHPrivateKey = args[i]
			case a == "--generate-key":
				o.GenerateKey = true
			case a == "--version" && i+1 < len(args):
				i++; o.Version = args[i]
			case a == "--sni" && i+1 < len(args):
				i++; o.SNI = args[i]
			case a == "--tg-token" && i+1 < len(args):
				i++; o.TelegramToken = args[i]
			case a == "--tg-admin" && i+1 < len(args):
				i++; o.TelegramAdminID = args[i]
			case a == "--skip-install":
				o.SkipInstall = true
			}
		}
		if o.Host == "" {
			fmt.Fprintln(os.Stderr, "required: --host")
			os.Exit(2)
		}
		if o.SSHPrivateKey == "" {
			o.GenerateKey = true
		}
		if err := deploy.DeployPrimary(o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "secondary":
	o := deploy.SecondaryOpts{PrimaryUser: "root", SecondaryUser: "root", SNI: "api.vk.me"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--primary" && i+1 < len(args):
				i++; o.PrimaryHost = args[i]
			case a == "--primary-key" && i+1 < len(args):
				i++; o.PrimaryKey = args[i]
			case a == "--host" && i+1 < len(args):
				i++; o.SecondaryHost = args[i]
			case a == "--password" && i+1 < len(args):
				i++; o.SecondaryPass = args[i]
			case a == "--user" && i+1 < len(args):
				i++; o.SecondaryUser = args[i]
			case a == "--sni" && i+1 < len(args):
				i++; o.SNI = args[i]
			}
		}
		if o.PrimaryHost == "" || o.PrimaryKey == "" || o.SecondaryHost == "" {
			fmt.Fprintln(os.Stderr, "required: --primary --primary-key --host --password")
			os.Exit(2)
		}
		if err := deploy.DeploySecondary(o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "edge":
	o := deploy.EdgeOpts{PrimaryUser: "root", RouterUser: "root", Version: "0.8.1", AgentArch: "arm64"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--primary" && i+1 < len(args):
				i++; o.PrimaryHost = args[i]
			case a == "--primary-key" && i+1 < len(args):
				i++; o.PrimaryKey = args[i]
			case a == "--router" && i+1 < len(args):
				i++; o.RouterHost = args[i]
			case a == "--password" && i+1 < len(args):
				i++; o.RouterPass = args[i]
			case a == "--id" && i+1 < len(args):
				i++; o.DeviceID = args[i]
			case a == "--server" && i+1 < len(args):
				i++; o.ServerURL = args[i]
			case a == "--arch" && i+1 < len(args):
				i++; o.AgentArch = args[i]
			}
		}
		if o.RouterHost == "" || o.DeviceID == "" {
			fmt.Fprintln(os.Stderr, "required: --router --id [--primary --primary-key --arch]")
			os.Exit(2)
		}
		if err := deploy.DeployEdge(o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("edge provisioned; approve on primary: netductor edge pending / edge approve", o.DeviceID)
	default:
		fmt.Fprintln(os.Stderr, "unknown deploy target", args[0])
		os.Exit(2)
	}
	_ = strings.TrimSpace
}
