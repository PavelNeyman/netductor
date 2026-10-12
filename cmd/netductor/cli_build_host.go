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
  netductor build-host enable|disable|clear`)
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
