package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/cli18n"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
)

// binaryRole is set at link time: node | operator | all (dev default).
var binaryRole = "all"

var version = "0.9.0"

func isOperatorSurface() bool { return binaryRole == "all" || binaryRole == "operator" }
func isNodeSurface() bool     { return binaryRole == "all" || binaryRole == "node" }

func rejectSurface(needOp, needNode bool, cmd string) {
	if binaryRole == "all" {
		return
	}
	if needOp && !isOperatorSurface() {
		fmt.Fprintf(os.Stderr, "%s: operator-only command - use netductor-op on the workstation (Mac)\n", cmd)
		os.Exit(2)
	}
	if needNode && !isNodeSurface() {
		fmt.Fprintf(os.Stderr, "%s: node-only command - on the VPS use the node binary (netductor-linux-*)\n", cmd)
		os.Exit(2)
	}
}

func main() {
	ndconfig.Load()

	if len(os.Args) < 2 {
		if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			runTUI(nil)
			return
		}
		printHelp()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "version", "-v", "--version":
		role := binaryRole
		if role == "" {
			role = "all"
		}
		fmt.Printf("netductor %s (%s)\n", version, role)
	case "tui", "menu":
		runTUI(os.Args[2:])
	case "help", "-h", "--help":
		printHelp()
	case "doctor":
		rejectSurface(false, true, "doctor")
		if len(os.Args) > 2 && os.Args[2] == "--legacy" {
			fmt.Fprintln(os.Stderr, "legacy doctor removed")
			os.Exit(2)
			return
		}
		os.Exit(runDoctorNative())
	case "vpn":
		rejectSurface(false, true, "vpn")
		runVPN(os.Args[2:])
	case "ssh-hosts", "known-hosts":
		rejectSurface(false, true, "ssh-hosts")
		runSSHHosts(os.Args[2:])
	case "sites":
		rejectSurface(false, true, "sites")
		runSites(os.Args[2:])
	case "nodes":
		rejectSurface(false, true, "nodes")
		runNodes(os.Args[2:])
	case "mtls":
		rejectSurface(false, true, "mtls")
		runMTLS(os.Args[2:])
	case "tls":
		rejectSurface(false, true, "tls")
		runTLS(os.Args[2:])
	case "redirect-serve", "import-redirect":
		rejectSurface(false, true, "redirect-serve")
		runRedirectServe(os.Args[2:])
	case "secondary":
		rejectSurface(false, true, "secondary")
		runSecondary(os.Args[2:])
	case "addons", "addon":
		rejectSurface(false, true, "addons")
		runAddons(os.Args[2:])
	case "git":
		rejectSurface(false, true, "git")
		os.Exit(runGit(os.Args[2:]))
	case "registry":
		rejectSurface(false, true, "registry")
		os.Exit(runRegistry(os.Args[2:]))
	case "ci":
		rejectSurface(false, true, "ci")
		os.Exit(runCI(os.Args[2:]))
	case "edge":
		rejectSurface(false, true, "edge")
		runEdgeCLI(os.Args[2:])
	case "nvr":
		rejectSurface(false, true, "nvr")
		runNVR(os.Args[2:])
	case "status":
		rejectSurface(false, true, "status")
		runStatus()
	case "install":
		rejectSurface(false, true, "install")
		runInstall(os.Args[2:])
	case "probe":
		rejectSurface(false, true, "probe")
		runProbe(os.Args[2:])
	case "collect":
		rejectSurface(false, true, "collect")
		os.Exit(runCollect())
	case "audit":
		rejectSurface(false, true, "audit")
		n := 50
		if len(os.Args) > 2 && os.Args[2] == "tail" && len(os.Args) > 3 {
			fmt.Sscanf(os.Args[3], "%d", &n)
		}
		for _, ev := range audit.Tail(n) {
			fmt.Printf("%d\t%s\t%s\t%s\t%s\n", ev.TS, ev.Actor, ev.Action, ev.Target, ev.Detail)
		}
	case "operator":
		rejectSurface(true, false, "operator")
		runOperator(os.Args[2:])
	case "domain":
		runDomain(os.Args[2:])
	case "credentials":
		rejectSurface(true, false, "credentials")
		runCredentials(os.Args[2:])
	case "recovery":
		rejectSurface(false, true, "recovery")
		runRecovery(os.Args[2:])
	case "backup":
		runBackupCmd(os.Args[2:])
	case "deploy":
		rejectSurface(true, false, "deploy")
		runDeploy(os.Args[2:])
	case "fleet":
		runFleet(os.Args[2:])
	case "restore":
		rejectSurface(false, true, "restore")
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
		rejectSurface(false, true, "recover")
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
			fmt.Fprintln(os.Stderr, "   or: netductor recover --from-secondary http://SECONDARY:8790 --recovery-token TOKEN --key KEY (required; offline)")
			os.Exit(2)
		}
		if err := install.Recover(arch, key); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("recovered")
	case "self-install":
		rejectSurface(false, true, "self-install")
		runSelfInstall()
	case "update":
		rejectSurface(false, true, "update")
		runUpdate(true)
	case "serve":
		rejectSurface(false, true, "serve")
		runServe(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printHelp() {
	switch binaryRole {
	case "operator":
		fmt.Print("netductor-op — operator workstation (Mac/PC)\n\n")
		fmt.Print("  deploy primary|secondary|fleet|edge   bootstrap nodes over SSH\n")
		fmt.Print("  operator serve                        localhost Fleet WebUI (:7373)\n")
		fmt.Print("  credentials collect                   secrets -> ~/.netductor/credentials\n")
		fmt.Print("  tui|menu                              Setup wizard / fleet UI\n")
		fmt.Print("  version | help\n\n")
		fmt.Print("Node plane runs on the VPS as binary netductor (asset netductor-linux-*).\n")
		fmt.Print("Deploy downloads that node asset automatically (never netductor-op).\n\n")
		fmt.Print("Docs: docs/ARCHITECTURE-OPERATOR.md · docs/DEPLOY-MAC.md\n")
	case "node":
		fmt.Print("netductor — node control plane (VPS / primary / secondary)\n\n")
		fmt.Print("  install | serve | doctor | status | vpn | secondary | edge | mtls | ...\n")
		fmt.Print("  redirect-serve | domain | fleet | backup | restore | nvr | sites | ...\n\n")
		fmt.Print("Operator deploy UI is netductor-op on the workstation (not this binary).\n")
	default:
		fmt.Print(cli18n.T("help.main"))
		fmt.Print("\n  (dev build: binaryRole=all — both operator and node commands)\n")
	}
}
