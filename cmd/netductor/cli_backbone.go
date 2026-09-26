package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/backbone"
)

func runBackbone(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: netductor backbone init-primary|export|init-secondary|apply|status|conf\n")
		os.Exit(2)
	}
	switch args[0] {
	case "init-primary":
		port := backbone.DefaultPort
		ep := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--port" && i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i++
			} else if args[i] == "--endpoint" && i+1 < len(args) {
				ep = args[i+1]
				i++
			} else if !strings.HasPrefix(args[i], "-") && ep == "" {
				ep = args[i]
			}
		}
		s, err := backbone.InitPrimary(port, ep)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("backbone primary initialized port=%d endpoint=%s\n", s.Port, s.Endpoint)
		fmt.Printf("conf: %s/%s.conf\n", backbone.Dir(), backbone.InterfaceName)
		fmt.Println("next: netductor backbone export > secondary-backbone.json")
		fmt.Println("      netductor backbone apply")
	case "export":
		raw, err := backbone.ExportForSecondary()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Stdout.Write(append(raw, '\n'))
	case "init-secondary":
		ep := ""
		path := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--endpoint" && i+1 < len(args) {
				ep = args[i+1]
				i++
			} else if args[i] == "--file" && i+1 < len(args) {
				path = args[i+1]
				i++
			}
		}
		if path == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor backbone init-secondary --file export.json --endpoint PRIMARY_IP[:port]")
			os.Exit(2)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var st backbone.State
		if err := json.Unmarshal(raw, &st); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := backbone.InitSecondaryFromState(&st, ep); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("backbone secondary configured")
		fmt.Println("next: netductor backbone apply")
	case "apply":
		msg, err := backbone.Apply()
		if msg != "" {
			fmt.Println(msg)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("backbone up:", backbone.InterfaceName)
	case "status":
		r := backbone.Status()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
	case "conf":
		s, err := backbone.LoadState()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		path := backbone.Dir() + "/" + backbone.InterfaceName + ".conf"
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("# role=%s endpoint=%s\n%s", s.Role, s.Endpoint, b)
	default:
		fmt.Fprintln(os.Stderr, "unknown backbone subcommand")
		os.Exit(2)
	}
}
