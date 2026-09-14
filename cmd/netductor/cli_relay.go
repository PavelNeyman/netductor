package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/relay"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runRelay(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor relay export|join|links|status|sync|exit|provision --host --user --password")
		os.Exit(2)
	}
	switch args[0] {
	case "export":
		out := "bundle.json"
		sni := vpn.DefaultRealitySNI
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-o", "--output":
				if i+1 < len(args) {
					i++
					out = args[i]
				}
			case "--sni":
				if i+1 < len(args) {
					i++
					sni = args[i]
				}
			}
		}
		b, err := vpn.ExportRelayBundle(sni)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if id, tok, err := relay.IssueToken("relay"); err == nil {
			b.AgentID = id
			b.AgentToken = tok
			b.CoreAgentURL = "http://" + b.CoreIP + ":8788"
		}
		raw, _ := json.MarshalIndent(b, "", "  ")
		if err := os.WriteFile(out, append(raw, '\n'), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", out)
		fmt.Println("Copy to RU VPS and run: netductor relay join", out)
		fmt.Println("uplink user:", vpn.RelayUplinkName, "uuid="+b.UplinkUUID)
	case "join":
		path := "bundle.json"
		if len(args) > 1 {
			path = args[1]
		}
		if err := install.InstallRelay(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "links":
		dir := filepath.Join(paths.StateDir(), "relay", "clients")
		ents, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "no relay client links — run join on RU VPS first")
			os.Exit(1)
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			fmt.Printf("## %s\n%s\n", e.Name(), string(b))
		}
	case "prune":
		n := relay.PruneDuplicates()
		s := relay.PruneStale(10 * time.Minute)
		fmt.Printf("duplicates_removed=%d stale_removed=%d\n", n, s)
		// drop offline relay nodes from registry
		for _, d := range relay.List() {
			_ = d
		}
		nodes.MarkStaleRelays(180)
		if list, err := nodes.List(); err == nil {
			for _, n := range list {
				if n.Role == "relay" && n.Status == "offline" && n.PublicIP == "" {
					_ = nodes.Delete(n.ID)
				}
			}
		}
	case "remove":
		if len(args) < 2 {
			os.Exit(2)
		}
		if err := relay.RemoveDevice(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = nodes.Delete(args[1])
		fmt.Println("removed", args[1])
	case "agent":
		tokB, _ := os.ReadFile("/etc/netductor/secrets/relay_agent_token")
		urlB, _ := os.ReadFile("/etc/netductor/secrets/relay_core_url")
		tok := strings.TrimSpace(string(tokB))
		url := strings.TrimSpace(string(urlB))
		if len(args) > 1 && args[1] != "" {
			// optional overrides
		}
		if tok == "" || url == "" {
			fmt.Fprintln(os.Stderr, "missing relay_agent_token or relay_core_url")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "relay agent →", url)
		relay.AgentLoop(url, tok, 30*time.Second)
	case "provision":
		host, user, pass, sni := "", "root", "", ""
		port := 22
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--host" && i+1 < len(args):
				i++; host = args[i]
			case a == "--user" && i+1 < len(args):
				i++; user = args[i]
			case a == "--password" && i+1 < len(args):
				i++; pass = args[i]
			case a == "--port" && i+1 < len(args):
				i++; fmt.Sscanf(args[i], "%d", &port)
			case a == "--sni" && i+1 < len(args):
				i++; sni = args[i]
			}
		}
		if host == "" || pass == "" {
			fmt.Fprintln(os.Stderr, "required: --host and --password")
			os.Exit(2)
		}
		sni = vpn.ResolveRelaySNI(sni, host)
		fmt.Fprintln(os.Stderr, "provision SNI:", sni)
		b, err := vpn.ExportRelayBundle(sni)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if id, tok, err := relay.IssueToken("relay"); err == nil {
			b.AgentID = id
			b.AgentToken = tok
			b.CoreAgentURL = "http://" + b.CoreIP + ":8788"
		}
		raw, _ := json.MarshalIndent(b, "", "  ")
		res, err := relay.ProvisionFromCore(relay.ProvisionIn{
			Host: host, Port: port, User: user, Password: pass, SNI: sni,
		}, string(raw))
		if res != nil {
			fmt.Println(res.Log)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// Persist preferred SNI for next re-provision (not ya.ru hardcode).
		_ = vpn.RememberRelaySNI(sni)
		// Post-provision automation (was manual): prune stale same-IP relays, peer backup, refresh links.
		postProvisionRelay(host, sni)
		fmt.Println("provisioned", host)
	case "device":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor relay device <id>")
			os.Exit(2)
		}
		for _, d := range relay.List() {
			if d.ID != args[1] {
				continue
			}
			on := "offline"
			if relay.Online(d, 2*time.Minute) {
				on = "online"
			}
			fmt.Println("status:", on)
			fmt.Println("sb=", d.SingBoxOK, "cpu=", d.CPUPercent, "mem=", d.MemUsedMB, "/", d.MemTotalMB, "load=", d.Load1)
			if len(d.PendingCmds) > 0 {
				fmt.Println("pending:", d.PendingCmds)
			}
			if d.LastCmd != "" {
				ok := "fail"
				if d.LastCmdOK {
					ok = "ok"
				}
				fmt.Println("last_cmd:", d.LastCmd, ok, d.LastCmdAt.Format(time.RFC3339))
				if d.LastCmdLog != "" {
					log := d.LastCmdLog
					if len(log) > 1500 {
						log = log[len(log)-1500:]
					}
					fmt.Println(log)
				}
			}
			return
		}
		fmt.Println("not found")
	case "cmd":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor relay cmd <id> <reboot|upgrade|metrics>")
			os.Exit(2)
		}
		if err := relay.EnqueueCmd(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("queued", args[2], "for", args[1])
	case "sync":
		ver := relay.BumpConfigVer()
		fmt.Println("config_ver", ver)
		fmt.Println("relays will pull on next heartbeat (~30s)")
	case "exit":
		if len(args) < 2 || args[1] == "status" {
			fmt.Println("exit_enabled", relay.ExitEnabled())
			return
		}
		on := args[1] == "on" || args[1] == "1" || args[1] == "true"
		if args[1] == "off" || args[1] == "0" || args[1] == "false" {
			on = false
		} else if args[1] != "on" && args[1] != "1" && args[1] != "true" {
			fmt.Fprintln(os.Stderr, "usage: netductor relay exit on|off")
			os.Exit(2)
		}
		if err := relay.SetExitEnabled(on); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = vpn.ApplyConfig()
		fmt.Println("exit_enabled", on)
		fmt.Println("sing-box re-applied on core; relays will sync on next agent poll")
	case "status":
		_ = relay.PruneDuplicates()
		devs := relay.List()
		if len(devs) == 0 {
			b, err := os.ReadFile(filepath.Join(paths.StateDir(), "relay", "bundle.json"))
			if err != nil {
				fmt.Println("no relays registered")
				return
			}
			fmt.Println(string(b))
			return
		}
		for _, d := range devs {
			on := "offline"
			if relay.Online(d, 2*time.Minute) {
				on = "online"
			}
			fmt.Printf("%s	%s	%s	ip=%s	sb=%v	ver=%d\n", d.ID, d.Name, on, d.PublicIP, d.SingBoxOK, d.ConfigVer)
		}
		fmt.Println("config_ver", relay.ConfigVer())
	default:
		fmt.Fprintln(os.Stderr, "unknown relay subcommand")
		os.Exit(2)
	}
}


// postProvisionRelay finishes what operators used to do by hand after provision.
func postProvisionRelay(host, sni string) {
	fmt.Println("==> post-provision: prune stale relays")
	_ = relay.PruneDuplicates()
	_ = relay.PruneStale(10 * time.Minute)
	if list, err := nodes.List(); err == nil {
		for _, n := range list {
			if n.Role != "relay" {
				continue
			}
			if n.PublicIP == host && n.Status == "offline" {
				_ = nodes.Delete(n.ID)
				fmt.Println("removed stale offline node", n.ID)
			}
		}
	}
	// Preferred hostname if only one relay for this IP
	if list, err := nodes.List(); err == nil {
		for _, n := range list {
			if n.Role == "relay" && n.PublicIP == host && n.Status == "online" {
				_, _ = nodes.SetDesiredHostname(n.ID, "nd-relay-ru")
			}
		}
	}
	target := "root@" + host + ":/var/lib/netductor/backups/peers/core/"
	fmt.Println("==> post-provision: backup peer", target)
	_ = install.SetBackupPeer(target, "-o StrictHostKeyChecking=accept-new -o BatchMode=yes")
	// Ensure remote peer dir + push latest local backup (key auth after provision)
	_ = execSSHHost(host, "mkdir -p /var/lib/netductor/backups/peers/core")
	dir := filepath.Join(paths.StateDir(), "backups")
	ents, _ := os.ReadDir(dir)
	var latest string
	var latestMod time.Time
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".ndenc") && !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if latest == "" || info.ModTime().After(latestMod) {
			latest = filepath.Join(dir, name)
			latestMod = info.ModTime()
		}
	}
	if latest != "" {
		fmt.Println("==> post-provision: scp backup", latest)
		_ = execLocal("scp", "-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes", latest, target)
	}
	fmt.Println("==> post-provision: refresh client links")
	if n, err := vpn.RefreshLinks(""); err != nil {
		fmt.Println("refresh-links:", err)
	} else {
		fmt.Println("refresh-links: refreshed", n)
	}
	fmt.Println("==> post-provision: SNI remembered as", sni)
}

func execSSHHost(host, cmd string) error {
	c := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes", "root@"+host, cmd)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

func execLocal(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}
