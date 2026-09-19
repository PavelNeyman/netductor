package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
			_ = exec.Command("bash", "-c", "nohup bash -c 'sleep 3; systemctl reboot' >/dev/null 2>&1 &").Run()
			fmt.Println("queued")
		case "upgrade":
			logf := "/var/lib/netductor/core-upgrade.log"
			sh := "/var/lib/netductor/core-upgrade.sh"
			_ = os.MkdirAll("/var/lib/netductor", 0o755)
			_ = os.WriteFile(logf, append([]byte("started"), 10), 0o600)
			raw, _ := exec.Command("bash", "-c", "echo IyEvYmluL2Jhc2gKc2V0IC14CmV4cG9ydCBERUJJQU5fRlJPTlRFTkQ9bm9uaW50ZXJhY3RpdmUKYXB0LWdldCB1cGRhdGUgLXFxIDI+JjEgfCB0YWlsIC01CmFwdC1nZXQgLXkgLW8gRHBrZzo6T3B0aW9uczo6PS0tZm9yY2UtY29uZmRlZiAtbyBEcGtnOjpPcHRpb25zOjo9LS1mb3JjZS1jb25mb2xkIHVwZ3JhZGUgMj4mMSB8IHRhaWwgLTQwCndnZXQgLXFPIC90bXAvbmQuYmluIGh0dHBzOi8vZ2l0aHViLmNvbS9QYXZlbE5leW1hbi9uZXRkdWN0b3IvcmVsZWFzZXMvZG93bmxvYWQvdjAuNy4wLWRldi9uZXRkdWN0b3ItbGludXgtYW1kNjQKd2dldCAtcU8gL3RtcC9uZC10Zy5iaW4gaHR0cHM6Ly9naXRodWIuY29tL1BhdmVsTmV5bWFuL25ldGR1Y3Rvci9yZWxlYXNlcy9kb3dubG9hZC92MC43LjAtZGV2L25ldGR1Y3Rvci10Zy1saW51eC1hbWQ2NApjcCAvdG1wL25kLmJpbiAvdXNyL2xvY2FsL2Jpbi9uZXRkdWN0b3IKaW5zdGFsbCAtbSA3NTUgL3RtcC9uZC10Zy5iaW4gL29wdC9uZXRkdWN0b3IvYmluL25ldGR1Y3Rvci10ZwppbnN0YWxsIC1tIDc1NSAvdG1wL25kLXRnLmJpbiAvdXNyL2xvY2FsL2Jpbi9uZXRkdWN0b3ItdGcKc3lzdGVtY3RsIHJlc3RhcnQgc2luZy1ib3ggbmV0ZHVjdG9yLWFwaSAyPiYxIHx8IHRydWUKZWNobyBDT1JFX1VQR1JBREVfRE9ORQpUT0s9JChjYXQgL2V0Yy9uZXRkdWN0b3Ivc2VjcmV0cy90ZWxlZ3JhbV9ib3RfdG9rZW4gMj4vZGV2L251bGwpCkNIQVQ9JChjYXQgL2V0Yy9uZXRkdWN0b3Ivc2VjcmV0cy90ZWxlZ3JhbV9hZG1pbl9pZCAyPi9kZXYvbnVsbCkKaWYgWyAtbiAiJFRPSyIgXSAmJiBbIC1uICIkQ0hBVCIgXTsgdGhlbgogIGN1cmwgLXNTIC1YIFBPU1QgImh0dHBzOi8vYXBpLnRlbGVncmFtLm9yZy9ib3Qke1RPS30vc2VuZE1lc3NhZ2UiIC1kIGNoYXRfaWQ9IiRDSEFUIiAtLWRhdGEtdXJsZW5jb2RlICJ0ZXh0PeKchSB1cGdyYWRlIMK3IG5kLWNvcmUgZG9uZSIgPi9kZXYvbnVsbCAyPiYxIHx8IHRydWUKZmkKc3lzdGVtY3RsIHJlc3RhcnQgbmV0ZHVjdG9yLXRlbGVncmFtLWJvdCAyPiYxIHx8IHRydWUK | base64 -d").Output()
			_ = os.WriteFile(sh, raw, 0o700)
			_ = exec.Command("bash", "-c", "nohup bash "+sh+" >>"+logf+" 2>&1 &").Run()
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
