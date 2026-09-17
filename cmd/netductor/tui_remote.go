package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (m *model) hasRemote() bool {
	return strings.TrimSpace(m.remoteHost) != ""
}

func (m *model) remoteLabel() string {
	if !m.hasRemote() {
		return ""
	}
	u := orDefault(m.remoteUser, "root")
	return u + "@" + m.remoteHost
}

func (m *model) sshBaseArgs() []string {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
	}
	if m.remoteKey != "" {
		args = append(args, "-i", m.remoteKey)
	}
	if m.remotePassword != "" {
		// password auth: disable pubkey-only batch if using sshpass wrapper
		args = append(args, "-o", "PreferredAuthentications=password", "-o", "PubkeyAuthentication=no")
	} else {
		args = append(args, "-o", "BatchMode=yes")
	}
	return args
}

func (m *model) runNetductor(args ...string) string {
	if m.hasRemote() {
		u := orDefault(m.remoteUser, "root")
		target := u + "@" + m.remoteHost
		remoteCmd := append([]string{"netductor"}, args...)
		var cmd *exec.Cmd
		if m.remotePassword != "" {
			if _, err := exec.LookPath("sshpass"); err == nil {
				sshArgs := append(m.sshBaseArgs(), target)
				sshArgs = append(sshArgs, remoteCmd...)
				cmd = exec.Command("sshpass", append([]string{"-p", m.remotePassword, "ssh"}, sshArgs...)...)
			} else {
				return "sshpass not installed — use SSH key, or: brew install sshpass / apt install sshpass\n"
			}
		} else {
			sshArgs := append(m.sshBaseArgs(), target)
			sshArgs = append(sshArgs, remoteCmd...)
			cmd = exec.Command("ssh", sshArgs...)
		}
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

func loadRemoteFromEnv() (host, user string) {
	host = strings.TrimSpace(os.Getenv("NETDUCTOR_REMOTE"))
	user = strings.TrimSpace(os.Getenv("NETDUCTOR_REMOTE_USER"))
	if user == "" {
		user = "root"
	}
	return host, user
}
