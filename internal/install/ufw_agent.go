package install

import (
	"fmt"
	"os"
	"os/exec"
)

// ApplyAgentFirewall: deny plain :8788, allow mTLS :8789 from anywhere.
// Auth is mTLS client certificates (edge behind ISP NAT needs no IP filter).
func ApplyAgentFirewall() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil
	}
	_ = run("ufw", "delete", "allow", "8788/tcp")
	_ = run("ufw", "deny", "8788/tcp")
	_ = run("ufw", "delete", "allow", "8789/tcp")
	if err := run("ufw", "allow", "8789/tcp"); err != nil {
		return fmt.Errorf("ufw allow 8789: %w", err)
	}
	fmt.Fprintln(os.Stderr, "ufw: :8788 denied, :8789 allowed (mTLS required)")
	return nil
}

// RestrictAgentMTLSToIP kept as no-op for secondary provision call sites.
func RestrictAgentMTLSToIP(ip string) error {
	return ApplyAgentFirewall()
}

// AllowAgentMTLSFromIP no-op (allowlist removed).
func AllowAgentMTLSFromIP(ip string) error {
	return ApplyAgentFirewall()
}
