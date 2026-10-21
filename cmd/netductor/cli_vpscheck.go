package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/PavelNeyman/netductor/internal/vpscheck"
)

func runVPSCheck(args []string) {
	profile := "generic"
	primary := ""
	jsonOut := false
	skip := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--profile" && i+1 < len(args):
			i++
			profile = args[i]
		case a == "--primary" && i+1 < len(args):
			i++
			primary = args[i]
		case a == "--json":
			jsonOut = true
		case a == "--skip-install":
			skip = true
		case a == "-h" || a == "--help":
			fmt.Fprintln(os.Stderr, `usage: netductor vps-check [--profile primary|secondary|generic] [--primary IP] [--json] [--skip-install]
Runs local VPS scorecard (sysbench, iperf, IP region, HTTP/TCP). Installs curl/iperf3/sysbench via apt when needed.`)
			os.Exit(0)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	rep, err := vpscheck.RunLocal(ctx, vpscheck.Options{
		Profile:     vpscheck.ParseProfile(profile),
		PrimaryIP:   primary,
		SkipInstall: skip,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "vps-check:", err)
		os.Exit(1)
	}
	if jsonOut {
		fmt.Println(rep.JSON())
	} else {
		fmt.Print(rep.Text())
	}
	if rep.Fail > 0 {
		os.Exit(2)
	}
}
