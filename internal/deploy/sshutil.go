package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func lookSSHPass() (string, error) {
	p, err := exec.LookPath("sshpass")
	if err != nil {
		return "", fmt.Errorf("sshpass not found (Mac: brew install hudochenkov/sshpass/sshpass or use key-only after first bootstrap)")
	}
	return p, nil
}

func sshOpts(keyPath string, passwordAuth bool) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=20",
	}
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	if passwordAuth {
		args = append(args, "-o", "PreferredAuthentications=password", "-o", "PubkeyAuthentication=no")
	} else {
		args = append(args, "-o", "BatchMode=yes")
	}
	return args
}

// runSSH: password is passed via SSHPASS env + sshpass -e (never argv -p).
func runSSH(password, keyPath, user, host string, remoteCmd string) (string, error) {
	target := user + "@" + host
	usePass := password != "" && (keyPath == "" || !fileExists(keyPath))
	base := sshOpts(keyPath, usePass)
	if usePass {
		sp, err := lookSSHPass()
		if err != nil {
			return "", err
		}
		args := append([]string{"-e", "ssh"}, base...)
		args = append(args, target, remoteCmd)
		cmd := exec.Command(sp, args...)
		cmd.Env = append(os.Environ(), "SSHPASS="+password)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	args := append(base, target, remoteCmd)
	out, err := exec.Command("ssh", args...).CombinedOutput()
	return string(out), err
}

func runSCP(password, keyPath, user, host, local, remotePath string) error {
	target := user + "@" + host + ":" + remotePath
	usePass := password != "" && (keyPath == "" || !fileExists(keyPath))
	base := sshOpts(keyPath, usePass)
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
	args := append(base, local, target)
	out, err := exec.Command("scp", args...).CombinedOutput()
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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := home + "/.ssh"
	_ = os.MkdirAll(dir, 0o700)
	return dir + "/netductor_primary", nil
}

// ShellQuote for safe remote argv (not for secrets in process list).
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
