package fleet

import (
	"fmt"
	"os"
	"os/exec"
)

// ApplyLampac installs Lampac on this host only (intended: primary).
// Remote deploy to secondary was removed (secondary = VPN entry only).
func ApplyLampac() error {
	fmt.Fprintln(os.Stderr, "==> lampac on local host (primary); secondary is VPN-entry only")
	cmd := exec.Command("netductor", "install", "lampac")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install lampac: %w", err)
	}
	return nil
}
