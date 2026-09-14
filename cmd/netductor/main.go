package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/install"
)

var version = "0.7.0-dev"

func main() {
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
	case "relay":
		runRelay(os.Args[2:])
	case "addons", "addon":
		runAddons(os.Args[2:])
	case "edge":
		runEdgeCLI(os.Args[2:])
	case "status":
		runStatus()
	case "install":
		runInstall(os.Args[2:])
	case "probe":
		runProbe(os.Args[2:])
	case "collect":
		os.Exit(runCollect())
	case "backup":
		runBackupCmd()
	case "restore":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor restore <archive>")
			os.Exit(2)
		}
		if err := install.Restore(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("restored")
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

  tui|menu [--mode vps|openwrt|workstation|operator]
  backup | self-install | update
  version | doctor | status | vpn | sites | ssh-hosts | relay | addons | edge | serve | install | probe | collect | help

  (no args on a TTY → interactive menu)

serve:
  --bind ADDR   (default 127.0.0.1)
  --port PORT   (default 8787)
  --tls-cert PATH --tls-key PATH
  --no-proxy
`)
}


