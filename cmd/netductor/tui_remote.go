package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Remote target: run netductor on a VPS from workstation TUI over SSH.

func (m *model) hasRemote() bool {
	return strings.TrimSpace(m.remoteHost) != ""
}

func (m *model) remoteLabel() string {
	if !m.hasRemote() {
		return ""
	}
	u := m.remoteUser
	if u == "" {
		u = "root"
	}
	return u + "@" + m.remoteHost
}

// runNetductor runs local `netductor args` or `ssh user@host netductor args`.
func (m *model) runNetductor(args ...string) string {
	if m.hasRemote() {
		u := m.remoteUser
		if u == "" {
			u = "root"
		}
		target := u + "@" + m.remoteHost
		sshArgs := []string{
			"-o", "BatchMode=yes",
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ConnectTimeout=15",
			target,
			"netductor",
		}
		sshArgs = append(sshArgs, args...)
		cmd := exec.Command("ssh", sshArgs...)
		out, err := cmd.CombinedOutput()
		s := string(out)
		if err != nil {
			s += fmt.Sprintf("\n[ssh %s] %v\n", target, err)
		}
		return s
	}
	cmd := exec.Command("netductor", args...)
	out, err := cmd.CombinedOutput()
	s := string(out)
	if err != nil {
		s += fmt.Sprintf("\n[%v]\n", err)
	}
	return s
}

func (m *model) showCmd(args ...string) {
	prefix := "local"
	if m.hasRemote() {
		prefix = m.remoteLabel()
	}
	body := m.runNetductor(args...)
	m.output = fmt.Sprintf("→ %s: netductor %s\n\n%s", prefix, strings.Join(args, " "), body)
	m.screen = screenOutput
}

// default remote from env (operator laptop).
func loadRemoteFromEnv() (host, user string) {
	host = strings.TrimSpace(os.Getenv("NETDUCTOR_REMOTE"))
	user = strings.TrimSpace(os.Getenv("NETDUCTOR_REMOTE_USER"))
	if user == "" {
		user = "root"
	}
	return host, user
}
