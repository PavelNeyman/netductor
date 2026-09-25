package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
)

// Node control-plane binary (VPS). No deploy/TUI/operator serve.
var version = "0.9.12"

func main() {
	ndconfig.Load()

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Printf("netductor %s (node)\n", version)
	case "help", "-h", "--help":
		printHelp()
	case "doctor":
		if len(os.Args) > 2 && os.Args[2] == "--legacy" {
			fmt.Fprintln(os.Stderr, "legacy doctor removed")
			os.Exit(2)
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
	case "secondary":
		runSecondary(os.Args[2:])
	case "addons", "addon":
		runAddons(os.Args[2:])
	case "git":
		os.Exit(runGit(os.Args[2:]))
	case "registry":
		os.Exit(runRegistry(os.Args[2:]))
	case "ci":
		os.Exit(runCI(os.Args[2:]))
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
		for _, e := range audit.Tail(n) {
			fmt.Println(e)
		}
	case "domain":
		runDomain(os.Args[2:])
	case "recovery":
		runRecovery(os.Args[2:])
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
			fmt.Fprintln(os.Stderr, "usage: netductor restore --key KEY (required; offline) <archive.ndenc|tar.gz>")
			os.Exit(2)
		}
		if err := install.Restore(arch, key); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("restored")
	case "recover":
		key, arch, fromSec, recTok := "", "", "", ""
		for i := 2; i < len(os.Args); i++ {
			a := os.Args[i]
			if a == "--key" && i+1 < len(os.Args) {
				i++
				key = os.Args[i]
				continue
			}
			if a == "--from-secondary" && i+1 < len(os.Args) {
				i++
				fromSec = os.Args[i]
				continue
			}
			if a == "--recovery-token" && i+1 < len(os.Args) {
				i++
				recTok = os.Args[i]
				continue
			}
			if !strings.HasPrefix(a, "-") && arch == "" {
				arch = a
			}
		}
		if fromSec != "" {
			if err := install.RecoverFromSecondary(fromSec, recTok, key); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("recovered from secondary")
			return
		}
		if arch == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor recover --key KEY (required; offline) <archive.ndenc>")
			fmt.Fprintln(os.Stderr, "   or: netductor recover --from-secondary http://SECONDARY:8790 --recovery-token TOKEN --key KEY")
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
	case "deploy", "operator", "credentials", "tui", "menu":
		fmt.Fprintf(os.Stderr, "%s: not available on the node binary — use netductor-op on the workstation\n", os.Args[1])
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "unknown: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`netductor — node control plane (VPS / primary / secondary)

  install | serve | doctor | status | vpn | secondary | edge | mtls | nvr
  redirect-serve | domain | fleet | backup | restore | sites | nodes | …
  version | help

Workstation deploy / TUI: install netductor-op (not this binary).
`)
}
