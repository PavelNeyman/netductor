package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runUplinkMux(args []string) {
	path := filepath.Join(paths.EtcDir(), "uplink_mux_mode")
	if len(args) == 0 || args[0] == "status" || args[0] == "show" {
		fmt.Println(vpn.UplinkMuxMode())
		return
	}
	if args[0] == "set" && len(args) > 1 {
		mode := strings.ToLower(strings.TrimSpace(args[1]))
		switch mode {
		case "on", "off", "h2mux":
		default:
			fmt.Fprintln(os.Stderr, "mode: on | off | h2mux")
			os.Exit(2)
		}
		_ = os.MkdirAll(paths.EtcDir(), 0o755)
		if err := os.WriteFile(path, []byte(mode+"\n"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("uplink_mux_mode=", mode)
		fmt.Println("re-apply secondary sing-box config and restart sing-box for effect")
		return
	}
	fmt.Fprintln(os.Stderr, "usage: netductor uplink-mux status|set on|off|h2mux")
	os.Exit(2)
}
