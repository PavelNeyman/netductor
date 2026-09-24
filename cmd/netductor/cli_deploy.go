package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
	"github.com/PavelNeyman/netductor/internal/operator"
)

func runDeploy(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage: netductor deploy primary|secondary|edge [flags]

  primary   — bootstrap Debian VPS from this machine (Mac/PC)
  secondary — provision RU entry from Mac (prepare-pack on primary; Mac SSHs to secondary)
  edge      — OpenWrt agent (+ optional guest/network, same as TUI)

primary:
  --host --password [--user root] [--generate-key] [--key-passphrase]
  [--key PATH] [--tg-token] [--tg-admin] [--sni] [--domain-base] [--le-email]
  [--with-lampac] [--with-git-registry]

secondary:
  --primary --primary-key --host --password [--user] [--sni]
  [--primary-key-passphrase] [--secondary-key]

edge:
  --router --id --password [--user root] [--arch arm64]
  --primary --primary-key [--primary-key-passphrase] [--server https://IP:8789]
  --guest [--guest-ssid] [--guest-pin] [--guest-psk]
  --configure-net
  --lan-ip --lan-mask --dhcp-start --dhcp-limit
  --wifi-ssid-24 --wifi-key-24 [--wifi-ssid-5] [--wifi-key-5]
  --wan-proto dhcp|static|pppoe
  --wan-ip --wan-mask --wan-gateway --wan-dns
  --pppoe-user --pppoe-pass [--pppoe-service] [--pppoe-ac]

See: netductor tui → Setup wizard`)
		os.Exit(2)
	}
	switch args[0] {
	case "primary":
		o := operator.PrimarySpec{Version: deploy.Release, SNI: "api.vk.me", User: "root"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--host" && i+1 < len(args):
				i++
				o.Host = args[i]
			case a == "--user" && i+1 < len(args):
				i++
				o.User = args[i]
			case a == "--password" && i+1 < len(args):
				i++
				o.Password = args[i]
			case a == "--key" && i+1 < len(args):
				i++
				o.SSHPrivateKey = args[i]
			case a == "--generate-key":
				o.GenerateKey = true
			case a == "--with-lampac":
				o.WithLampac = true
			case a == "--with-git-registry":
				o.WithGitRegistry = true
			case a == "--key-passphrase" && i+1 < len(args):
				i++
				o.KeyPassphrase = args[i]
			case a == "--version" && i+1 < len(args):
				i++
				o.Version = args[i]
			case a == "--sni" && i+1 < len(args):
				i++
				o.SNI = args[i]
			case a == "--tg-token" && i+1 < len(args):
				i++
				o.TelegramToken = args[i]
			case a == "--tg-admin" && i+1 < len(args):
				i++
				o.TelegramAdminID = args[i]
			case a == "--skip-install":
				o.SkipInstall = true
			case a == "--domain-base" && i+1 < len(args):
				i++
				o.DomainBase = args[i]
			case a == "--le-email" && i+1 < len(args):
				i++
				o.DomainEmail = args[i]
				o.DomainLE = true
			case a == "--domain-http":
				o.DomainHTTP = true
			case a == "--cf-proxy", a == "--cloudflare":
				o.DomainCFProxy = true
			}
		}
		if o.Host == "" {
			fmt.Fprintln(os.Stderr, "required: --host")
			os.Exit(2)
		}
		if o.SSHPrivateKey == "" {
			o.GenerateKey = true
		}
		if err := operator.DeployPrimary(o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "secondary":
		o := operator.SecondarySpec{PrimaryUser: "root", SecondaryUser: "root", SNI: "api.vk.me"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--primary" && i+1 < len(args):
				i++
				o.PrimaryHost = args[i]
			case a == "--primary-key" && i+1 < len(args):
				i++
				o.PrimaryKey = args[i]
			case a == "--host" && i+1 < len(args):
				i++
				o.SecondaryHost = args[i]
			case a == "--password" && i+1 < len(args):
				i++
				o.SecondaryPass = args[i]
			case a == "--user" && i+1 < len(args):
				i++
				o.SecondaryUser = args[i]
			case a == "--sni" && i+1 < len(args):
				i++
				o.SNI = args[i]
			case a == "--primary-key-passphrase" && i+1 < len(args):
				i++
				o.PrimaryKeyPassphrase = args[i]
			case a == "--secondary-key" && i+1 < len(args):
				i++
				o.SecondarySSHKey = args[i]
			}
		}
		if o.PrimaryHost == "" || o.PrimaryKey == "" || o.SecondaryHost == "" {
			fmt.Fprintln(os.Stderr, "required: --primary --primary-key --host (--password or --secondary-key)")
			os.Exit(2)
		}
		if err := operator.DeploySecondary(o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "fleet":
		f := operator.FleetSpec{DoPrimary: true, DoSecondary: true}
		f.Primary = operator.PrimarySpec{User: "root", SNI: "api.vk.me", Version: deploy.Release, GenerateKey: true}
		f.Secondary = operator.SecondarySpec{PrimaryUser: "root", SecondaryUser: "root", SNI: "api.vk.me"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--primary-host" && i+1 < len(args):
				i++
				f.Primary.Host = args[i]
			case a == "--primary-password" && i+1 < len(args):
				i++
				f.Primary.Password = args[i]
			case a == "--secondary-host" && i+1 < len(args):
				i++
				f.Secondary.SecondaryHost = args[i]
			case a == "--secondary-password" && i+1 < len(args):
				i++
				f.Secondary.SecondaryPass = args[i]
			case a == "--key" && i+1 < len(args):
				i++
				f.Primary.SSHPrivateKey = args[i]
				f.Primary.GenerateKey = false
			case a == "--sni" && i+1 < len(args):
				i++
				f.Primary.SNI = args[i]
				f.Secondary.SNI = args[i]
			case a == "--domain-base" && i+1 < len(args):
				i++
				f.Primary.DomainBase = args[i]
			case a == "--le-email" && i+1 < len(args):
				i++
				f.Primary.DomainEmail = args[i]
			case a == "--with-lampac":
				f.Primary.WithLampac = true
			case a == "--primary-only":
				f.DoSecondary = false
			case a == "--secondary-only":
				f.DoPrimary = false
			}
		}
		if f.DoPrimary && f.Primary.Host == "" {
			fmt.Fprintln(os.Stderr, "fleet: --primary-host required")
			os.Exit(2)
		}
		if f.DoSecondary && f.Secondary.SecondaryHost == "" {
			fmt.Fprintln(os.Stderr, "fleet: --secondary-host required")
			os.Exit(2)
		}
		if err := operator.FleetDeploy(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "edge":
		o := deploy.EdgeOpts{PrimaryUser: "root", RouterUser: "root", Version: deploy.Release, AgentArch: "arm64"}
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--primary" && i+1 < len(args):
				i++
				o.PrimaryHost = args[i]
			case a == "--primary-key" && i+1 < len(args):
				i++
				o.PrimaryKey = args[i]
			case a == "--primary-key-passphrase" && i+1 < len(args):
				i++
				o.PrimaryKeyPassphrase = args[i]
			case a == "--router" && i+1 < len(args):
				i++
				o.RouterHost = args[i]
			case a == "--user" && i+1 < len(args):
				i++
				o.RouterUser = args[i]
			case a == "--password" && i+1 < len(args):
				i++
				o.RouterPass = args[i]
			case a == "--id" && i+1 < len(args):
				i++
				o.DeviceID = args[i]
			case a == "--server" && i+1 < len(args):
				i++
				o.ServerURL = args[i]
			case a == "--arch" && i+1 < len(args):
				i++
				o.AgentArch = args[i]
			case a == "--guest":
				o.GuestEnable = true
			case a == "--guest-ssid" && i+1 < len(args):
				i++
				o.GuestSSID = args[i]
			case a == "--guest-pin" && i+1 < len(args):
				i++
				o.GuestPIN = args[i]
			case a == "--guest-psk" && i+1 < len(args):
				i++
				o.GuestPSK = args[i]
			case a == "--configure-net":
				o.NetConfigure = true
			case a == "--lan-ip" && i+1 < len(args):
				i++
				o.LANIP = args[i]
				o.NetConfigure = true
			case a == "--lan-mask" && i+1 < len(args):
				i++
				o.LANMask = args[i]
			case a == "--dhcp-start" && i+1 < len(args):
				i++
				o.DHCPStart = args[i]
			case a == "--dhcp-limit" && i+1 < len(args):
				i++
				o.DHCPLimit = args[i]
			case a == "--wifi-ssid-24" && i+1 < len(args):
				i++
				o.WiFiSSID24 = args[i]
				o.NetConfigure = true
			case a == "--wifi-key-24" && i+1 < len(args):
				i++
				o.WiFiKey24 = args[i]
			case a == "--wifi-ssid-5" && i+1 < len(args):
				i++
				o.WiFiSSID5 = args[i]
			case a == "--wifi-key-5" && i+1 < len(args):
				i++
				o.WiFiKey5 = args[i]
			case a == "--wifi-ssid" && i+1 < len(args):
				i++
				o.WiFiSSID = args[i]
				o.NetConfigure = true
			case a == "--wifi-key" && i+1 < len(args):
				i++
				o.WiFiKey = args[i]
			case a == "--wan-proto" && i+1 < len(args):
				i++
				o.WANProto = args[i]
				o.NetConfigure = true
			case a == "--wan-ip" && i+1 < len(args):
				i++
				o.WANIP = args[i]
			case a == "--wan-mask" && i+1 < len(args):
				i++
				o.WANMask = args[i]
			case a == "--wan-gateway" && i+1 < len(args):
				i++
				o.WANGateway = args[i]
			case a == "--wan-dns" && i+1 < len(args):
				i++
				o.WANDNS = args[i]
			case a == "--pppoe-user" && i+1 < len(args):
				i++
				o.PPPoEUser = args[i]
			case a == "--pppoe-pass" && i+1 < len(args):
				i++
				o.PPPoEPass = args[i]
			case a == "--pppoe-service" && i+1 < len(args):
				i++
				o.PPPoEService = args[i]
			case a == "--pppoe-ac" && i+1 < len(args):
				i++
				o.PPPoEAC = args[i]
			}
		}
		if o.RouterPass == "" {
			o.RouterPass = os.Getenv("NETDUCTOR_SSH_PASSWORD")
		}
		if o.RouterHost == "" || o.DeviceID == "" {
			fmt.Fprintln(os.Stderr, "required: --router --id --password (or NETDUCTOR_SSH_PASSWORD)")
			os.Exit(2)
		}
		if o.ServerURL == "" && o.PrimaryHost != "" {
			o.ServerURL = "https://" + o.PrimaryHost + ":8789"
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
