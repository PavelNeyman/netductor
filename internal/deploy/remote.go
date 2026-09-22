package deploy

import "fmt"

// RunOnPrimary executes a remote command on primary via SSH (operator key).
func RunOnPrimary(host, user, keyPath, keyPassphrase, cmd string) (string, error) {
	if host == "" || keyPath == "" {
		return "", fmt.Errorf("primary host and key required")
	}
	if user == "" {
		user = "root"
	}
	return runSSH("", keyPath, user, host, cmd, keyPassphrase)
}
