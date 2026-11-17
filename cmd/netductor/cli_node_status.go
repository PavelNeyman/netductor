package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/controlplane"
	"github.com/PavelNeyman/netductor/internal/version"
)

func runNodeStatus(args []string) {
	hn, _ := os.Hostname()
	fmt.Printf("hostname=%s version=%s\n", hn, version.Release)
	fmt.Printf("control_role=%s write_allowed=%v is_leader=%v\n",
		controlplane.Get(), controlplane.WriteAllowed(), controlplane.WriteAllowed())
	for _, u := range []string{"netductor-api", "netductor-telegram-bot", "sing-box", "blocky", "netductor-redirect"} {
		b, _ := exec.Command("systemctl", "is-active", u).Output()
		fmt.Printf("  %s=%s\n", u, strings.TrimSpace(string(b)))
	}
}
