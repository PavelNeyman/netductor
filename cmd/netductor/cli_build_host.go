package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func runBuildHost(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor build-host show [--json]
  netductor build-host set --host IP [--port 52222] [--user root] [--key PATH] [--label NAME] [--enable|--disable]
  netductor build-host test
  netductor build-host enable|disable|clear
  netductor build-host deploy --host IP [--pass PASS] [--port 22] [--user root] [--key PATH] [--label NAME]
  netductor build-host list [--json]
  netductor build-host watch`)
		return 2
	}
	switch args[0] {
	case "show", "status":
		c := gitstore.LoadBuildHost()
		if len(args) > 1 && args[1] == "--json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(c)
			return 0
		}
		fmt.Println(gitstore.BuildHostStatusLine())
		if c.SSHHost != "" {
			fmt.Printf("user=%s host=%s port=%d key=%s remote=%s label=%s enabled=%v\n",
				c.SSHUser, c.SSHHost, c.SSHPort, c.IdentityFile, c.RemoteDir, c.Label, c.Enabled)
		}
		return 0
	case "test":
		out, err := gitstore.TestBuildHost()
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok")
		return 0
	case "enable":
		c := gitstore.LoadBuildHost()
		if c.SSHHost == "" {
			fmt.Fprintln(os.Stderr, "set --host first")
			return 1
		}
		c.Enabled = true
		if err := gitstore.SaveBuildHost(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("enabled")
		return 0
	case "disable":
		c := gitstore.LoadBuildHost()
		c.Enabled = false
		if err := gitstore.SaveBuildHost(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("disabled")
		return 0
	case "clear":
		_ = os.Remove(strings.TrimSpace(os.Getenv("NETDUCTOR_BUILD_HOST_CONFIG")))
		// default path
		c := gitstore.BuildHostConfig{}
		_ = gitstore.SaveBuildHost(c)
		fmt.Println("cleared (disabled empty config)")
		return 0
	case "list":
		list := gitstore.ListBuildHostStates()
		asJSON := false
		for _, a := range args[1:] {
			if a == "--json" {
				asJSON = true
			}
		}
		if asJSON {
			for i := range list {
				list[i].Token = ""
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(list)
			return 0
		}
		if len(list) == 0 {
			fmt.Println("(no registered build hosts)")
			return 0
		}
		for _, s := range list {
			on := "offline"
			if s.Online {
				on = "online"
			}
			fmt.Printf("%s\t%s\t%s\t%s@%s:%d\tdocker=%v\tdisk=%.1fG\tagent=%s\n",
				s.ID, s.Label, on, s.SSHUser, s.SSHHost, s.SSHPort, s.Docker, s.DiskFreeGB, s.AgentVersion)
		}
		return 0
	case "watch":
		gitstore.WatchBuildHosts()
		fmt.Println("probed")
		return 0
	case "deploy":
		var o gitstore.DeployBuildHostOpts
		o.Port = 22
		o.User = "root"
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--host":
				if i+1 < len(args) {
					o.Host = args[i+1]
					i++
				}
			case "--pass", "--password":
				if i+1 < len(args) {
					o.Password = args[i+1]
					i++
				}
			case "--port":
				if i+1 < len(args) {
					o.Port, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--user":
				if i+1 < len(args) {
					o.User = args[i+1]
					i++
				}
			case "--key":
				if i+1 < len(args) {
					o.KeyPath = args[i+1]
					i++
				}
			case "--label":
				if i+1 < len(args) {
					o.Label = args[i+1]
					i++
				}
			case "--primary-url":
				if i+1 < len(args) {
					o.PrimaryURL = args[i+1]
					i++
				}
			}
		}
		if o.Host == "" {
			fmt.Fprintln(os.Stderr, "--host required")
			return 2
		}
		if o.KeyPath == "" {
			if _, err := os.Stat("/root/.ssh/id_ed25519"); err == nil {
				o.KeyPath = "/root/.ssh/id_ed25519"
			}
		}
		log, err := gitstore.DeployBuildHost(o)
		fmt.Print(log)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "set":
		c := gitstore.LoadBuildHost()
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--host":
				if i+1 < len(args) {
					c.SSHHost = args[i+1]
					i++
				}
			case "--port":
				if i+1 < len(args) {
					c.SSHPort, _ = strconv.Atoi(args[i+1])
					i++
				}
			case "--user":
				if i+1 < len(args) {
					c.SSHUser = args[i+1]
					i++
				}
			case "--key":
				if i+1 < len(args) {
					c.IdentityFile = args[i+1]
					i++
				}
			case "--label":
				if i+1 < len(args) {
					c.Label = args[i+1]
					i++
				}
			case "--remote-dir":
				if i+1 < len(args) {
					c.RemoteDir = args[i+1]
					i++
				}
			case "--enable":
				c.Enabled = true
			case "--disable":
				c.Enabled = false
			}
		}
		if c.SSHHost == "" {
			fmt.Fprintln(os.Stderr, "--host required")
			return 2
		}
		if err := gitstore.SaveBuildHost(c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(gitstore.BuildHostStatusLine())
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown build-host subcommand")
		return 2
	}
}
