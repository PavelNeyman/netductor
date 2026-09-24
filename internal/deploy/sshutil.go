package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)


func sshPort() string {
	if p := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT")); p != "" {
		return p
	}
	return "22"
}


// clearHostKeys drops stale known_hosts entries after VPS reinstall (changed host key).
func clearHostKeys(host string) {
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
		args = append(args, "-i", keyPath)
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

// runSSH: host password via SSHPASS+sshpass -e; key passphrase via SSH_ASKPASS (never argv).
func runSSH(password, keyPath, user, host, remoteCmd, keyPassphrase string) (string, error) {
	clearHostKeys(host)
	target := user + "@" + host
	usePass := password != "" && (keyPath == "" || !fileExists(keyPath))
	base := sshOpts(keyPath, usePass, !usePass && keyPassphrase != "")
	base = append([]string{"-p", sshPort()}, base...)
	if usePass {
		if sp, err := lookSSHPass(); err == nil {
			args := append([]string{"-e", "ssh"}, base...)
			args = append(args, target, remoteCmd)
			cmd := exec.Command(sp, args...)
			cmd.Env = append(os.Environ(), "SSHPASS="+password)
			out, err := cmd.CombinedOutput()
			return string(out), err
		}
		ask, cleanup, err := writeAskPass(password)
		if err != nil {
			return "", fmt.Errorf("sshpass missing and ASKPASS failed: %w", err)
		}
		defer cleanup()
		args := append(append([]string{}, base...), target, remoteCmd)
		cmd := exec.Command("setsid", append([]string{"ssh"}, args...)...)
		cmd.Env = append(os.Environ(), "SSH_ASKPASS="+ask, "SSH_ASKPASS_REQUIRE=force", "DISPLAY=.")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	env, cleanup, err := sshEnvWithAskPass(keyPassphrase)
	if err != nil {
		return "", err
	}
	defer cleanup()
	args := append(base, target, remoteCmd)
	cmd := exec.Command("ssh", args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runSCP(password, keyPath, user, host, local, remotePath, keyPassphrase string) error {
	target := user + "@" + host + ":" + remotePath
	usePass := password != "" && (keyPath == "" || !fileExists(keyPath))
	base := sshOpts(keyPath, usePass, !usePass && keyPassphrase != "")
	base = append([]string{"-P", sshPort()}, base...)
	if usePass {
		sp, err := lookSSHPass()
		if err != nil {
			return err
		}
		args := append([]string{"-e", "scp"}, base...)
		args = append(args, local, target)
		cmd := exec.Command(sp, args...)
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
	args := append(base, local, target)
	cmd := exec.Command("scp", args...)
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
func scpTo(keyPath, localPath, remoteSpec, keyPassphrase string) error {
	base := []string{"-P", sshPort(), "-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes"}
	if keyPath != "" {
		base = append(base, "-i", keyPath)
	}
	args := append(base, localPath, remoteSpec)
	cmd := exec.Command("scp", args...)
	env, cleanup, err := sshEnvWithAskPass(keyPassphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
