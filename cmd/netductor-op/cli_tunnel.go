package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// SSH local forward to node API (127.0.0.1:8787 on remote → local port).
func runTunnel(args []string) {
	host, user, key, local, remote := "", "root", "", "8787", "127.0.0.1:8787"
	sshPort := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
	if sshPort == "" {
		sshPort = "52222"
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--host" && i+1 < len(args):
			i++
			host = args[i]
		case a == "--user" && i+1 < len(args):
			i++
			user = args[i]
		case a == "--key" && i+1 < len(args):
			i++
			key = args[i]
		case a == "--local-port" && i+1 < len(args):
			i++
			local = args[i]
		case a == "--help", a == "-h":
			fmt.Fprint(os.Stderr, `usage:
  netductor-op tunnel --host IP [--user root] [--key ~/.ssh/netductor_primary] [--local-port 8787]

Forwards local PORT → remote 127.0.0.1:8787 (node API). Leave running; open WebUI Control.
`)
			os.Exit(0)
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
	fwd := fmt.Sprintf("%s:%s", local, remote)
	argv := []string{
		"-N", "-L", fwd,
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-p", sshPort,
		"-i", key,
		"--", user + "@" + host,
	}
	fmt.Fprintf(os.Stderr, "tunnel: 127.0.0.1:%s → %s@%s %s (Ctrl+C stop)\n", local, user, host, remote)
	cmd := exec.Command("ssh", argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		_ = cmd.Process.Kill()
	}()
	err := cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
