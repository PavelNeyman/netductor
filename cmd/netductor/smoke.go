package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

func runSmoke(args []string) int {
	mode := "dual"
	if len(args) > 0 {
		mode = args[0]
	}
	bin := "netductor"
	if b, err := os.Executable(); err == nil && b != "" {
		bin = b
	}
	fail := 0
	check := func(name, cmd string, argv ...string) {
		out, err := exec.Command(cmd, argv...).CombinedOutput()
		line := strings.TrimSpace(string(out))
		if len(line) > 200 {
			line = line[:200] + "…"
		}
		if err != nil {
			fmt.Printf("FAIL %s: %v (%s)\n", name, err, line)
			fail++
			return
		}
		fmt.Printf("OK   %s: %s\n", name, line)
	}
	fmt.Printf("smoke mode=%s ts=%s\n", mode, time.Now().UTC().Format(time.RFC3339))
	check("version", bin, "version")
	check("api-health", "curl", "-sf", "--max-time", "3", "http://127.0.0.1:8787/health")
	if mode == "dual" || mode == "secondary" {
		list := secondary.List()
		if len(list) == 0 {
			fmt.Println("FAIL secondary: none registered")
			fail++
		} else {
			for _, d := range list {
				if !secondary.Online(d, 3*time.Minute) {
					fmt.Printf("FAIL secondary %s offline\n", d.ID)
					fail++
				} else {
					fmt.Printf("OK   secondary %s online\n", d.ID)
				}
			}
		}
		check("channel", bin, "channel", "status")
	}
	check("sing-box", "systemctl", "is-active", "sing-box")
	if fail > 0 {
		fmt.Printf("Summary: fail=%d\n", fail)
		return 1
	}
	fmt.Println("Summary: ok")
	return 0
}
