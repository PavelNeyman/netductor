package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runNodeCLI executes `netductor <args>` on the configured primary (SSH), or fails if unset.
func runNodeCLI(args []string) {
	s := loadTUISettings()
	host := strings.TrimSpace(s.RemoteHost)
	if host == "" {
		fmt.Fprintln(os.Stderr, "primary host not set — open TUI Setup / settings first")
		os.Exit(2)
	}
	user := orDefault(s.RemoteUser, "root")
	key := strings.TrimSpace(s.RemoteKey)
	target := user + "@" + host
	remote := append([]string{"netductor"}, args...)
	sshArgs := []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "ConnectTimeout=15", "-o", "BatchMode=yes"}
	if key != "" {
		sshArgs = append(sshArgs, "-i", key)
	}
	sshArgs = append(sshArgs, "--", target)
	sshArgs = append(sshArgs, remote...)
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
