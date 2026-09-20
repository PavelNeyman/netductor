package mikrotik

import (
	"fmt"
	"strings"
)

// HardenSSH installs operator pubkey for user and best-effort disables password login.
// RouterOS varies by version; key install is required, password-off is best-effort.
func HardenSSH(c ConnOpts, operatorPubKey string) error {
	pub := strings.TrimSpace(operatorPubKey)
	if pub == "" {
		return fmt.Errorf("operator pubkey required")
	}
	user := c.user()
	// Remove duplicate key lines if present, then add.
	// ROS accepts: /user ssh-keys add user=admin key="ssh-ed25519 AAAA... comment=netductor"
	addCmd := fmt.Sprintf(`/user ssh-keys add user=%s key="%s" comment=netductor-operator`, user, escapeROS(pub))
	if out, err := Run(c, addCmd); err != nil {
		// already exists is ok on some versions
		if !strings.Contains(strings.ToLower(out+err.Error()), "already") &&
			!strings.Contains(strings.ToLower(out), "exists") {
			return fmt.Errorf("ssh-keys add: %v %s", err, out)
		}
	}
	// Best-effort: prefer key auth (ROS 7+)
	for _, cmd := range []string{
		`/ip ssh set always-allow-password-login=no`,
		`/ip ssh set strong-crypto=yes`,
	} {
		_, _ = Run(c, cmd)
	}
	return nil
}

func escapeROS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
