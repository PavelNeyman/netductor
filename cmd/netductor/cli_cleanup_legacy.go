package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runCleanupLegacy(args []string) {
	dry := true
	for _, a := range args {
		if a == "--apply" {
			dry = false
		}
	}
	ifaces := []string{"nd-backbone", "nd-awg", "nd-wgios"}
	units := []string{
		"nd-backbone-iperf-s", "nd-backbone-iperf-c", "nd-awg-iperf-s", "nd-awg-iperf-c",
		"nd-backbone-soak", "wg-quick@nd-backbone", "wg-quick@nd-wgios",
		"strongswan", "strongswan-starter",
	}
	files := []string{
		"/etc/wireguard/nd-backbone.conf", "/etc/wireguard/nd-wgios.conf",
		"/etc/amnezia/amneziawg/nd-awg.conf",
	}
	fmt.Println("cleanup test/legacy (keeps nd-svc-sp / nd-svc-ps)")
	for _, u := range units {
		fmt.Println(" unit", u)
		if !dry {
			_ = exec.Command("systemctl", "stop", u).Run()
			_ = exec.Command("systemctl", "disable", u).Run()
			_ = exec.Command("systemctl", "reset-failed", u).Run()
		}
	}
	for _, iface := range ifaces {
		fmt.Println(" iface", iface)
		if !dry {
			_ = exec.Command("wg-quick", "down", iface).Run()
			_ = exec.Command("awg-quick", "down", iface).Run()
			_ = exec.Command("ip", "link", "delete", iface).Run()
		}
	}
	for _, f := range files {
		fmt.Println(" conf", f)
		if !dry {
			_ = os.Remove(f)
		}
	}
	if dry {
		fmt.Println("dry-run; pass --apply")
	} else {
		fmt.Println("done")
	}
}
