package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RestrictAgentMTLSToIP replaces open 8789/tcp with allow only from secondaryIP.
// Also denies plain agent plane 8788. Safe to call repeatedly.
func RestrictAgentMTLSToIP(secondaryIP string) error {
	secondaryIP = strings.TrimSpace(secondaryIP)
	if secondaryIP == "" {
		return fmt.Errorf("secondary IP required")
	}
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = run("ufw", "delete", "allow", "8789/tcp")
	_ = run("ufw", "delete", "allow", "8788/tcp")
	_ = run("ufw", "deny", "8788/tcp")
	if err := run("ufw", "allow", "from", secondaryIP, "to", "any", "port", "8789", "proto", "tcp"); err != nil {
		return fmt.Errorf("ufw allow 8789 from %s: %w", secondaryIP, err)
	}
	fmt.Fprintf(os.Stderr, "ufw: agent mTLS :8789 allowed only from %s\n", secondaryIP)
	return nil
}
