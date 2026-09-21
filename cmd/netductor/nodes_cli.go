package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/deploy"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func syncRelaysIntoNodes() {
	_ = secondary.PruneDuplicates()
	for _, d := range secondary.List() {
		host := d.Name
		if host == "" || host == "secondary" {
			host = "nd-secondary-" + strings.ReplaceAll(d.PublicIP, ".", "-")
		}
		st := "offline"
		if secondary.Online(d, 2*time.Minute) {
			st = "online"
		}
		ls := d.LastSeen.Unix()
		if ls <= 0 {
			ls = time.Now().Unix()
		}
		_, _ = nodes.UpsertFromDevice(nodes.Node{
			ID: d.ID, Hostname: host, Role: "secondary", Kind: "vps",
			PublicIP: d.PublicIP, Status: st, LastSeen: ls,
		})
	}
}

func runNodes(args []string) {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		syncRelaysIntoNodes()
		nodes.MarkStaleRelays(180)
		syncRelaysIntoNodes()
		list, err := nodes.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, n := range list {
			fmt.Printf("id=%s\thost=%s\trole=%s\tkind=%s\tip=%s\tstatus=%s\tdesired=%s\n",
				n.ID, n.Hostname, n.Role, n.Kind, n.PublicIP, n.Status, n.DesiredHN)
		}
	case "rename":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor nodes rename <id> <hostname>")
			fmt.Fprintln(os.Stderr, "  hostname format: nd-<role>-<marker>  e.g. nd-secondary-msk01")
			os.Exit(2)
		}
		syncRelaysIntoNodes()
		n, err := nodes.SetDesiredHostname(args[1], args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = secondary.Rename(args[1], args[2])
		fmt.Printf("desired_hostname=%s for %s\n", n.DesiredHN, n.ID)
	case "sync-local":
		if err := nodes.SyncLocalHostname(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "local-cmd":
		if len(args) < 2 {
			os.Exit(2)
		}
		switch args[1] {
		case "reboot":
			go func() { time.Sleep(3 * time.Second); _ = exec.Command("systemctl", "reboot").Run() }()
			fmt.Println("queued")
		case "upgrade":
			logf := "/var/lib/netductor/core-upgrade.log"
			_ = os.MkdirAll("/var/lib/netductor", 0o755)
			_ = os.WriteFile(logf, []byte("started\n"), 0o600)
			go func() {
				f, _ := os.OpenFile(logf, os.O_APPEND|os.O_WRONLY, 0o600)
				defer func() {
					if f != nil {
						f.Close()
					}
				}()
				logw := func(s string) {
					if f != nil {
						_, _ = f.WriteString(s + "\n")
					}
				}
				env := append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
				run := func(name string, args ...string) {
					cmd := exec.Command(name, args...)
					cmd.Env = env
					out, err := cmd.CombinedOutput()
					logw(string(out))
					if err != nil {
						logw(err.Error())
					}
				}
				run("apt-get", "update", "-qq")
				run("apt-get", "-y", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", "upgrade")
				ver := deploy.Release
				if ver == "" {
					ver = "0.8.22"
				}
				ver = strings.TrimPrefix(ver, "v")
				base := "https://github.com/PavelNeyman/netductor/releases/download/v" + ver
				run("wget", "-qO", "/tmp/nd.bin", base+"/netductor-linux-amd64")
				run("wget", "-qO", "/tmp/nd-tg.bin", base+"/netductor-tg-linux-amd64")
				if st, err := os.Stat("/tmp/nd.bin"); err == nil && st.Size() > 1000 {
					_ = exec.Command("install", "-m", "755", "/tmp/nd.bin", "/usr/local/bin/netductor").Run()
				}
				if st, err := os.Stat("/tmp/nd-tg.bin"); err == nil && st.Size() > 1000 {
					_ = exec.Command("install", "-m", "755", "/tmp/nd-tg.bin", "/opt/netductor/bin/netductor-tg").Run()
					_ = exec.Command("install", "-m", "755", "/tmp/nd-tg.bin", "/usr/local/bin/netductor-tg").Run()
				}
				_ = exec.Command("systemctl", "restart", "sing-box").Run()
				_ = exec.Command("systemctl", "restart", "netductor-api").Run()
				logw("CORE_UPGRADE_DONE")
				// restart bot last so this process can exit cleanly
				go func() {
					time.Sleep(2 * time.Second)
					_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Run()
				}()
			}()
			fmt.Println("queued")
		default:
			fmt.Println("unknown", args[1])
		}
	case "id":
		fmt.Println(nodes.LocalStableID())
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor nodes list|rename|sync-local|id")
		os.Exit(2)
	}
}
