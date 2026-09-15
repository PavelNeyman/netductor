package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func runMTLS(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Print("netductor mtls ensure  — generate CA/server/client certs for agent plane\n")
		return
	}
	switch args[0] {
	case "ensure":
		pub := ""
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_ip")); err == nil {
			pub = strings.TrimSpace(string(b))
		}
		if err := mtls.EnsureAll(pub); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("mtls ready:", mtls.Dir())
		fmt.Println("  server:", mtls.ServerReady(), " client:", mtls.ClientReady())
	default:
		fmt.Fprintln(os.Stderr, "unknown mtls subcommand")
		os.Exit(2)
	}
}
