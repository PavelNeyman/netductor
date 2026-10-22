package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/PavelNeyman/netductor/internal/vpscheck"
)

func runVPSCheckOp(args []string) {
	var host, user, pass, key, keyPass, profile, primary, port string
	user = "root"
	profile = "secondary"
	port = "22"
	jsonOut := false
	skip := false
	local := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case (a == "--host" || a == "--router") && i+1 < len(args):
			i++
			host = args[i]
		case a == "--user" && i+1 < len(args):
			i++
			user = args[i]
		case a == "--password" && i+1 < len(args):
			i++
			pass = args[i]
		case (a == "--key" || a == "--primary-key") && i+1 < len(args):
			i++
			key = args[i]
		case a == "--key-passphrase" && i+1 < len(args):
			i++
			keyPass = args[i]
		case a == "--profile" && i+1 < len(args):
			i++
			profile = args[i]
		case a == "--primary" && i+1 < len(args):
			i++
			primary = args[i]
		case a == "--port" && i+1 < len(args):
			i++
			port = args[i]
		case a == "--json":
			jsonOut = true
		case a == "--skip-install":
			skip = true
		case a == "--local":
			local = true
		case a == "-h" || a == "--help":
			fmt.Fprintln(os.Stderr, `usage:
  netductor-op vps-check --host IP --password PASS [--profile secondary|primary] [--primary CORE_IP]
  netductor-op vps-check --host IP --key PATH
  netductor-op vps-check --local [--profile …]

Also: job type vps.precheck (Web/TUI). Installs curl/iperf3/sysbench on the target when possible.`)
			os.Exit(0)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var rep vpscheck.Report
	var err error
	if local {
		rep, err = vpscheck.RunLocal(ctx, vpscheck.Options{
			Profile: vpscheck.ParseProfile(profile), PrimaryIP: primary, SkipInstall: skip,
		})
	} else {
		if host == "" {
			fmt.Fprintln(os.Stderr, "vps-check: --host or --local required")
			os.Exit(2)
		}
		// Structured JSON needs Report once — run SSH path only (no job + re-run).
		// Text mode: prefer job so Web/TUI share the same runner; no second SSH.
		if jsonOut {
			rep, err = vpscheck.RunViaSSH(ctx, vpscheck.SSHOpts{
				Host: host, User: user, Password: pass, KeyPath: key,
				KeyPassphrase: keyPass, Port: port,
				Profile: vpscheck.ParseProfile(profile), PrimaryIP: primary, SkipInstall: skip,
			})
		} else {
			params := map[string]any{
				"host": host, "user": user, "password": pass, "key": key,
				"key_passphrase": keyPass, "port": port, "profile": profile,
				"primary": primary, "skip_install": skip,
			}
			if _, jerr := startAndWaitJob("vps.precheck", params); jerr == nil {
				fmt.Println("vps-check: job finished (see log above)")
				return
			}
			// Fallback if job runner unavailable
			rep, err = vpscheck.RunViaSSH(ctx, vpscheck.SSHOpts{
				Host: host, User: user, Password: pass, KeyPath: key,
				KeyPassphrase: keyPass, Port: port,
				Profile: vpscheck.ParseProfile(profile), PrimaryIP: primary, SkipInstall: skip,
			})
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if jsonOut {
		fmt.Println(rep.JSON())
		return
	}
	fmt.Print(rep.Text())
}
