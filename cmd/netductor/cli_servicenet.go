package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/servicenet"
)

func runServiceNet(args []string) {
	if len(args) < 1 {
		fmt.Print(`netductor servicenet status [--json]
netductor servicenet apply
`)
		return
	}
	switch args[0] {
	case "status":
		st := servicenet.Collect()
		if len(args) > 1 && args[1] == "--json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(st)
			if !st.OK {
				os.Exit(1)
			}
			return
		}
		fmt.Printf("enabled=%v ok=%v iface=%s gateway=%s\n", st.Enabled, st.OK, st.Iface, st.Gateway)
		if len(st.Addrs) > 0 {
			fmt.Printf("addrs: %v\n", st.Addrs)
		}
		if st.Detail != "" {
			fmt.Println(st.Detail)
		}
		if !st.OK {
			os.Exit(1)
		}
	case "apply":
		if err := servicenet.Ensure(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		st := servicenet.Collect()
		fmt.Printf("servicenet apply ok=%v addrs=%v\n", st.OK, st.Addrs)
	default:
		fmt.Fprintln(os.Stderr, "unknown servicenet subcommand")
		os.Exit(2)
	}
}
