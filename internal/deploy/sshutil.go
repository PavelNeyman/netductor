package deploy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func sshPort() string {
	if p := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT")); p != "" {
		return p
	}
	return "22"
}

// day2SSHPort is SSH after harden (primary/secondary). Default 52222.
func day2SSHPort() string {
	if p := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT")); p != "" {
		return p
	}
	return "52222"
}

// factorySSHPort is first-login / OpenWrt stock SSH. Default 22.
func factorySSHPort() string {
	if p := strings.TrimSpace(os.Getenv("NETDUCTOR_ROUTER_SSH_PORT")); p != "" {
		return p
	}
	return "22"
}

// ClearHostKeys drops known_hosts for host — only for explicit reinstall/host-key rotation.
// Do NOT call on every SSH hop (that defeats TOFU / MITM detection).
func ClearHostKeys(host string) {
	if host == "" {
		return
	}
	_ = exec.Command("ssh-keygen", "-R", host).Run()
	_ = exec.Command("ssh-keygen", "-R", "["+host+"]:22").Run()
	_ = exec.Command("ssh-keygen", "-R", "["+host+"]:52222").Run()
	if p := sshPort(); p != "22" && p != "52222" {
		_ = exec.Command("ssh-keygen", "-R", "["+host+"]:"+p).Run()
	}
}

func lookSSHPass() (string, error) {
	p, err := exec.LookPath("sshpass")
	if err != nil {
		return "", fmt.Errorf("sshpass not found (Mac: brew install hudochenkov/sshpass/sshpass or use key-only after first bootstrap)")
	}
	return p, nil
}

func sshOpts(keyPath string, passwordAuth bool, keyPassphrase bool) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=20",
	}
	if keyPath != "" {
		args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
	}
	if passwordAuth {
		args = append(args, "-o", "PreferredAuthentications=password", "-o", "PubkeyAuthentication=no")
	} else if keyPassphrase {
		// BatchMode blocks passphrase prompts; ASKPASS supplies it instead
		args = append(args, "-o", "BatchMode=no", "-o", "NumberOfPasswordPrompts=1")
	} else {
		args = append(args, "-o", "BatchMode=yes")
	}
	return args
}

// writeAskPass returns a 0700 script that prints secret once (for SSH_ASKPASS).
func writeAskPass(secret string) (string, func(), error) {
	f, err := os.CreateTemp("", "nd-askpass-*.sh")
	if err != nil {
		return "", nil, err
	}
	path := f.Name()
	// Avoid shell metachar issues: write secret to sidecar file, script cats it
	secPath := path + ".secret"
	if err := os.WriteFile(secPath, []byte(secret), 0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, err
	}
	body := "#!/bin/sh\ncat '" + secPath + "'\n"
	if _, err := f.WriteString(body); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		_ = os.Remove(secPath)
		return "", nil, err
	}
	_ = f.Close()
	_ = os.Chmod(path, 0o700)
	cleanup := func() {
		_ = os.Remove(path)
		_ = os.Remove(secPath)
	}
	return path, cleanup, nil
}

func sshEnvWithAskPass(keyPass string) ([]string, func(), error) {
	env := os.Environ()
	if keyPass == "" {
		return env, func() {}, nil
	}
	ask, cleanup, err := writeAskPass(keyPass)
	if err != nil {
		return nil, nil, err
	}
	env = append(env,
		"SSH_ASKPASS="+ask,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=.", // required by some OpenSSH builds for ASKPASS
	)
	return env, cleanup, nil
}

// runSSH uses sshPort() (env NETDUCTOR_SSH_PORT or 22). Prefer runSSHOnPort for mixed primary/router hops.
func runSSH(password, keyPath, user, host, remoteCmd, keyPassphrase string) (string, error) {
	return runSSHOnPort(sshPort(), password, keyPath, user, host, remoteCmd, keyPassphrase)
}

// runSSHOnPort: host password via SSHPASS+sshpass -e; key passphrase via SSH_ASKPASS (never argv).
func runSSHOnPort(port, password, keyPath, user, host, remoteCmd, keyPassphrase string) (string, error) {
	if port == "" {
		port = sshPort()
	}
	target := user + "@" + host
	// Prefer password when provided (OpenWrt day-0 after setPass still needs password until harden).
	// Key-only when password is empty.
	usePass := password != ""
	base := sshOpts(keyPath, usePass, !usePass && keyPassphrase != "")
	base = append([]string{"-p", port}, base...)
	if usePass {
		if sp, err := lookSSHPass(); err == nil {
			args := append([]string{"-e", "ssh"}, base...)
			args = append(args, "--", target, remoteCmd)
			cmd := exec.CommandContext(currentSSHCtx(), sp, args...)
			cmd.Env = append(os.Environ(), "SSHPASS="+password)
			out, err := cmd.CombinedOutput()
			return string(out), err
		}
		ask, cleanup, err := writeAskPass(password)
		if err != nil {
			return "", fmt.Errorf("sshpass missing and ASKPASS failed: %w", err)
		}
		defer cleanup()
		args := append(append([]string{}, base...), "--", target, remoteCmd)
		cmd := exec.CommandContext(currentSSHCtx(), "setsid", append([]string{"ssh"}, args...)...)
		cmd.Env = append(os.Environ(), "SSH_ASKPASS="+ask, "SSH_ASKPASS_REQUIRE=force", "DISPLAY=.")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	env, cleanup, err := sshEnvWithAskPass(keyPassphrase)
	if err != nil {
		return "", err
	}
	defer cleanup()
	args := append(base, "--", target, remoteCmd)
	cmd := exec.CommandContext(currentSSHCtx(), "ssh", args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runSSHOnPortWithStdin is like runSSHOnPort but feeds stdin to the remote command.
// Secrets must travel on stdin, never in remoteCmd argv.
func runSSHOnPortWithStdin(port, password, keyPath, user, host, remoteCmd, keyPassphrase string, stdin []byte) (string, error) {
	if port == "" {
		port = sshPort()
	}
	target := user + "@" + host
	usePass := password != ""
	base := sshOpts(keyPath, usePass, !usePass && keyPassphrase != "")
	base = append([]string{"-p", port}, base...)
	run := func(cmd *exec.Cmd) (string, error) {
		cmd.Stdin = bytes.NewReader(stdin)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if usePass {
		if sp, err := lookSSHPass(); err == nil {
			args := append([]string{"-e", "ssh"}, base...)
			args = append(args, "--", target, remoteCmd)
			cmd := exec.CommandContext(currentSSHCtx(), sp, args...)
			cmd.Env = append(os.Environ(), "SSHPASS="+password)
			return run(cmd)
		}
		ask, cleanup, err := writeAskPass(password)
		if err != nil {
			return "", fmt.Errorf("sshpass missing and ASKPASS failed: %w", err)
		}
		defer cleanup()
		args := append(append([]string{}, base...), "--", target, remoteCmd)
		cmd := exec.CommandContext(currentSSHCtx(), "setsid", append([]string{"ssh"}, args...)...)
		cmd.Env = append(os.Environ(), "SSH_ASKPASS="+ask, "SSH_ASKPASS_REQUIRE=force", "DISPLAY=.")
		return run(cmd)
	}
	env, cleanup, err := sshEnvWithAskPass(keyPassphrase)
	if err != nil {
		return "", err
	}
	defer cleanup()
	args := append(base, "--", target, remoteCmd)
	cmd := exec.CommandContext(currentSSHCtx(), "ssh", args...)
	cmd.Env = env
	return run(cmd)
}

func runSCP(password, keyPath, user, host, local, remotePath, keyPassphrase string) error {
	return runSCPOnPort(sshPort(), password, keyPath, user, host, local, remotePath, keyPassphrase)
}

func runSCPOnPort(port, password, keyPath, user, host, local, remotePath, keyPassphrase string) error {
	if port == "" {
		port = sshPort()
	}
	target := user + "@" + host + ":" + remotePath
	usePass := password != "" // prefer password when set (router pre-harden)
	base := sshOpts(keyPath, usePass, !usePass && keyPassphrase != "")
	base = append([]string{"-P", port}, base...)
	if usePass {
		sp, err := lookSSHPass()
		if err != nil {
			return err
		}
		args := append([]string{"-e", "scp", "-O"}, base...)
		args = append(args, "--", local, target)
		cmd := exec.CommandContext(currentSSHCtx(), sp, args...)
		cmd.Env = append(os.Environ(), "SSHPASS="+password)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("scp: %s: %w", strings.TrimSpace(string(out)), err)
		}
		return nil
	}
	env, cleanup, err := sshEnvWithAskPass(keyPassphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	// -O: legacy SCP (Dropbear / OpenWrt have no SFTP subsystem).
	args := append(append([]string{"-O"}, base...), "--", local, target)
	cmd := exec.CommandContext(currentSSHCtx(), "scp", args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func defaultKeyPath() (string, error) {
	if p := os.Getenv("NETDUCTOR_KEY_PATH"); p != "" {
		_ = os.MkdirAll(filepath.Dir(p), 0o700)
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(dir, 0o700)
	return filepath.Join(dir, "netductor_primary"), nil
}

// ShellQuote for safe remote argv (not for secrets in process list).
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// scpTo copies localPath to remoteSpec (user@host:/path) using the same port/askpass rules as runSSH.

// waitEdgeSSH polls hosts with key-only SSH until one answers or timeout.
func waitEdgeSSH(hosts []string, keyPath, user, keyPassphrase string, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	deadline := time.Now().Add(timeout)
	port := factorySSHPort()
	for time.Now().Before(deadline) {
		for _, h := range hosts {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			if _, err := runSSHOnPort(port, "", keyPath, user, h, "echo nd-up", keyPassphrase); err == nil {
				return h
			}
		}
		time.Sleep(3 * time.Second)
	}
	return ""
}
