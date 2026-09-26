package main

import (
	"fmt"
	"os"
)

// Operator workstation binary (Mac/PC). Does not embed node plane (install/serve/vpn).
var version = "0.9.57"

func main() {
	if len(os.Args) < 2 {
		if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			runTUI(nil)
			return
		}
		printOpHelp()
		return
	}
	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Printf("netductor-op %s (operator)\n", version)
	case "help", "-h", "--help":
		printOpHelp()
	case "tui", "menu":
		runTUI(os.Args[2:])
	case "deploy":
		runDeploy(os.Args[2:])
	case "operator":
		runOperator(os.Args[2:])
	case "credentials":
		runCredentials(os.Args[2:])
	case "tunnel":
		runTunnel(os.Args[2:])
	case "session":
		runSession(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown: %s (operator binary — see netductor-op help)\n", os.Args[1])
		os.Exit(1)
	}
}

func printOpHelp() {
	fmt.Print(`netductor-op — operator workstation (Mac/PC)
  tunnel --host IP   # SSH local forward to node API


  deploy primary|secondary|fleet|edge   bootstrap nodes over SSH
  operator serve [--bind 127.0.0.1] [--port 7373] [--token SECRET]
  credentials collect                   secrets -> ~/.netductor/credentials
  tui|menu                              Setup wizard / fleet UI
  version | help

The VPS runs a separate binary: netductor (release asset netductor-linux-*).
Deploy downloads that node asset automatically — never this operator binary.

Docs: docs/ARCHITECTURE-OPERATOR.md · docs/DEPLOY-MAC.md · docs/BUILD-BINARIES.md
`)
}
