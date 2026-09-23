package deploy

import "fmt"

// RunOnPrimary executes a remote command via SSH with the operator key.
// Host may be primary or any VPS the key can reach (name kept for compatibility).
func RunOnPrimary(host, user, keyPath, keyPassphrase, cmd string) (string, error) {
	if host == "" || keyPath == "" {
		return "", fmt.Errorf("host and SSH key path required")
	}
	if user == "" {
		user = "root"
	}
	return runSSH("", keyPath, user, host, cmd, keyPassphrase)
}
