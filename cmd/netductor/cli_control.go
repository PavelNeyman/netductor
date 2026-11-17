package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/controlplane"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func runControl(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor control status
  netductor control sot-check
  netductor control endpoints
  netductor control token-init          # active: create control_standby_token
  netductor control sync-pull [--url URL] [--token TOK]
  netductor control restore-inbox [--key KEY] --confirm
  netductor control set-role active|standby --confirm
  netductor control promote --confirm`)
		os.Exit(2)
	}
	switch args[0] {
	case "status":
		r := controlplane.Get()
		fmt.Printf("control_role=%s write_allowed=%v file=%s\n", r, controlplane.WriteAllowed(), controlplane.RolePath())
		eps := controlplane.ReadEndpoints()
		if len(eps) == 0 {
			fmt.Println("control_endpoints: (none — optional " + controlplane.EndpointsPath() + ")")
		} else {
			fmt.Println("control_endpoints:")
			for _, e := range eps {
				fmt.Println(" ", e)
			}
		}
		fmt.Println("control_inbox:", controlplane.InboxDir())
		if arch, err := controlplane.LatestInbox(); err == nil {
			fmt.Println("inbox_latest:", arch)
		} else {
			fmt.Println("inbox_latest: (empty)")
		}
		if u := controlplane.SyncURL(); u != "" {
			fmt.Println("sync_url:", u)
		}
	case "sot-check", "check-sot":
		ok, lines := controlplane.CheckLocalSoT()
		for _, l := range lines {
			fmt.Println(l)
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "sot-check: FAIL (required paths missing — restore backup before promote)")
			os.Exit(1)
		}
		fmt.Println("sot-check: OK")
	case "endpoints":
		eps := controlplane.ReadEndpoints()
		for _, e := range eps {
			fmt.Println(e)
		}
		if len(eps) == 0 {
			fmt.Fprintln(os.Stderr, "# empty; write one URL per line to", controlplane.EndpointsPath())
		}
	case "token-init":
		path := filepath.Join(paths.EtcDir(), "secrets", "control_standby_token")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if b, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(b))) >= 16 {
			fmt.Println("control_standby_token: already set at", path)
			fmt.Fprintln(os.Stderr, "copy this file to standby: /etc/netductor/secrets/control_standby_token")
			return
		}
		tok := randomHexToken(32)
		if err := os.WriteFile(path, []byte(tok+"\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
		fmt.Fprintln(os.Stderr, "copy token to standby secrets/control_standby_token and set NETDUCTOR_CONTROL_SYNC_URL=https://ACTIVE:8789")
	case "sync-pull":
		url := controlplane.SyncURL()
		tok := controlplane.SyncToken()
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--url":
				if i+1 < len(args) {
					i++
					url = args[i]
				}
			case "--token":
				if i+1 < len(args) {
					i++
					tok = args[i]
				}
			}
		}
		if url == "" || tok == "" {
			fmt.Fprintln(os.Stderr, "need --url and --token (or NETDUCTOR_CONTROL_SYNC_URL + secrets/control_standby_token)")
			os.Exit(2)
		}
		path, name, sha, err := controlplane.PullLatest(url, tok, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("pulled %s (%s) sha256=%s\n", name, path, sha)
	case "restore-inbox":
		confirm := false
		key := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--confirm" || args[i] == "-y" {
				confirm = true
			}
			if args[i] == "--key" && i+1 < len(args) {
				i++
				key = args[i]
			}
		}
		if !confirm {
			fmt.Fprintln(os.Stderr, "usage: netductor control restore-inbox [--key KEY] --confirm")
			os.Exit(2)
		}
		if controlplane.Get() == controlplane.RoleActive {
			fmt.Fprintln(os.Stderr, "refusing restore-inbox on active control (set-role standby first, or use netductor recover)")
			os.Exit(2)
		}
		arch, err := controlplane.LatestInbox()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "restore-inbox:", arch)
		if err := install.Recover(arch, key); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("restore-inbox: recover done — next: netductor control promote --confirm (after fencing old active)")
	case "set-role":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor control set-role active|standby --confirm")
			os.Exit(2)
		}
		role := strings.ToLower(strings.TrimSpace(args[1]))
		confirm := false
		for _, a := range args[2:] {
			if a == "--confirm" || a == "-y" {
				confirm = true
			}
		}
		if !confirm {
			fmt.Fprintln(os.Stderr, "refusing without --confirm (avoids accidental split-brain)")
			os.Exit(2)
		}
		var r controlplane.Role
		switch role {
		case "active", "promote":
			r = controlplane.RoleActive
		case "standby", "passive":
			r = controlplane.RoleStandby
		default:
			fmt.Fprintf(os.Stderr, "unknown role %q\n", role)
			os.Exit(2)
		}
		if err := controlplane.SetRole(r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("control_role=%s write_allowed=%v\n", controlplane.Get(), controlplane.WriteAllowed())
		if r == controlplane.RoleActive {
			fmt.Fprintln(os.Stderr, "NOTE: ensure the previous active node is set to standby or powered off (fencing).")
		}
		if r == controlplane.RoleStandby {
			fmt.Fprintln(os.Stderr, "NOTE: mutating API returns 503 until promote; agents need SERVERS= this host only after promote.")
		}
	case "promote":
		confirm := false
		for _, a := range args[1:] {
			if a == "--confirm" || a == "-y" {
				confirm = true
			}
		}
		if !confirm {
			fmt.Fprintln(os.Stderr, "usage: netductor control promote --confirm")
			os.Exit(2)
		}
		ok, lines := controlplane.CheckLocalSoT()
		for _, l := range lines {
			fmt.Println(l)
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "promote aborted: sot-check FAIL — restore SoT from backup first")
			os.Exit(1)
		}
		if err := controlplane.SetRole(controlplane.RoleActive); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("promoted control_role=%s write_allowed=%v\n", controlplane.Get(), controlplane.WriteAllowed())
		fmt.Fprintln(os.Stderr, "Fence old active: netductor control set-role standby --confirm (on the old node), or power it off.")
	default:
		fmt.Fprintf(os.Stderr, "unknown control subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func randomHexToken(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
