package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
)

var version = "0.7.29-dev"

func main() {
	ndconfig.Load()

	if len(os.Args) < 2 {
		// interactive when terminal; else help
		if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			runTUI(nil)
			return
		}
		printHelp()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Printf("netductor %s\n", version)
	case "tui", "menu":
		runTUI(os.Args[2:])
	case "help", "-h", "--help":
		printHelp()
	case "doctor":
		if len(os.Args) > 2 && os.Args[2] == "--legacy" {
			fmt.Fprintln(os.Stderr, "legacy doctor removed"); os.Exit(2)
			return
		}
		os.Exit(runDoctorNative())
	case "vpn":
		runVPN(os.Args[2:])
	case "ssh-hosts", "known-hosts":
		runSSHHosts(os.Args[2:])
	case "sites":
		runSites(os.Args[2:])
	case "nodes":
		runNodes(os.Args[2:])
	case "mtls":
		runMTLS(os.Args[2:])
	case "tls":
		runTLS(os.Args[2:])
	case "redirect-serve", "import-redirect":
		runRedirectServe(os.Args[2:])
	case "secondary", "relay":
		runRelay(os.Args[2:])
	case "addons", "addon":
		runAddons(os.Args[2:])
	case "edge":
		runEdgeCLI(os.Args[2:])
	case "nvr":
		runNVR(os.Args[2:])
	case "status":
		runStatus()
	case "install":
		runInstall(os.Args[2:])
	case "probe":
		runProbe(os.Args[2:])
	case "collect":
		os.Exit(runCollect())
	case "audit":
		n := 50
		if len(os.Args) > 2 && os.Args[2] == "tail" && len(os.Args) > 3 {
			fmt.Sscanf(os.Args[3], "%d", &n)
		}
		for _, ev := range audit.Tail(n) {
			fmt.Printf("%d\t%s\t%s\t%s\t%s\n", ev.TS, ev.Actor, ev.Action, ev.Target, ev.Detail)
		}
	case "backup":
		runBackupCmd(os.Args[2:])
	case "fleet":
		runFleet(os.Args[2:])
	case "restore":
		key, arch := "", ""
		for i := 2; i < len(os.Args); i++ {
			a := os.Args[i]
			if a == "--key" && i+1 < len(os.Args) {
				i++
				key = os.Args[i]
				continue
			}
			if !strings.HasPrefix(a, "-") && arch == "" {
				arch = a
			}
		}
		if arch == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor restore [--key KEY] <archive.ndenc|tar.gz>")
			os.Exit(2)
		}
		if err := install.Restore(arch, key); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("restored")
	case "recover":
		key, arch := "", ""
		for i := 2; i < len(os.Args); i++ {
			a := os.Args[i]
			if a == "--key" && i+1 < len(os.Args) {
				i++
				key = os.Args[i]
				continue
			}
			if !strings.HasPrefix(a, "-") && arch == "" {
				arch = a
			}
		}
		if arch == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor recover [--key KEY] <archive.ndenc>")
			os.Exit(2)
		}
		if err := install.Recover(arch, key); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("recovered")
	case "self-install":
		runSelfInstall()
	case "update":
		runUpdate(true)
	case "serve":
		runServe(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown: %s\n", os.Args[1])
		os.Exit(1)
	}
}


func printHelp() {
	fmt.Print(`netductor — network control plane

  tui|menu [--mode vps|openwrt|workstation|operator] [--remote HOST] [--remote-user U] [--remote-key PATH] [--remote-password P]
  backup | restore | recover | fleet | audit | self-install | update
  version | doctor | status | vpn | sites | ssh-hosts | secondary | addons | edge | serve | install | probe | collect | help
  (alias: relay → secondary)

  (no args on a TTY → interactive menu)

serve:
  --bind ADDR   (default 127.0.0.1)
  --port PORT   (default 8787)
  --tls-cert PATH --tls-key PATH
  --no-proxy
`)
}


