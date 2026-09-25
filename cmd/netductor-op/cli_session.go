package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runSession(args []string) {
	if len(args) < 1 || args[0] == "help" {
		fmt.Fprint(os.Stderr, `usage:
  netductor-op session issue --host IP [--key PATH] [--hours 72]

SSH to node and run: netductor vpn session <hours>
Prints token (store in WebUI Settings → Node session).
`)
		os.Exit(2)
	}
	if args[0] != "issue" {
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		os.Exit(2)
	}
	host, key, hours := "", "", "72"
	user := "root"
	sshPort := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
	if sshPort == "" {
		sshPort = "52222"
	}
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--host" && i+1 < len(args):
			i++
			host = args[i]
		case a == "--key" && i+1 < len(args):
			i++
			key = args[i]
		case a == "--user" && i+1 < len(args):
			i++
			user = args[i]
		case a == "--hours" && i+1 < len(args):
			i++
			hours = args[i]
		}
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "required: --host")
		os.Exit(2)
	}
	if key == "" {
		home, _ := os.UserHomeDir()
		key = filepath.Join(home, ".ssh", "netductor_primary")
	}
	if strings.HasPrefix(key, "~/") {
		home, _ := os.UserHomeDir()
		key = filepath.Join(home, key[2:])
	}
	remote := fmt.Sprintf("netductor vpn session %s", hours)
	cmd := exec.Command("ssh", "-p", sshPort, "-i", key, "-o", "BatchMode=yes", "--", user+"@"+host, remote)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Fprintln(os.Stderr, string(ee.Stderr))
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tok := strings.TrimSpace(string(out))
	// first line only
	if i := strings.IndexByte(tok, '\n'); i >= 0 {
		tok = strings.TrimSpace(tok[:i])
	}
	fmt.Println(tok)
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".netductor")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "node_session"), []byte(tok+"\n"), 0o600)
	fmt.Fprintln(os.Stderr, "saved ~/.netductor/node_session — paste into WebUI Settings or use as Bearer")
}
