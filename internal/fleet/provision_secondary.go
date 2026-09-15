package fleet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/nodes"
)

// ProvisionSecondaryOpts operator input for full secondary deploy from primary.
type ProvisionSecondaryOpts struct {
	Host     string
	User     string
	Password string
	Port     int
	SNI      string
	NoLampac bool
	NoBotStandby bool
}

// ProvisionSecondary is the operator-facing path replacing ad-hoc "relay only":
//  1) VPN data-plane join (existing netductor relay provision)
//  2) fleet secondary + naming nd-secondary-*
//  3) backup peer + sync
//  4) optional Lampac on secondary
//  5) bot standby units on secondary (SOCKS→primary)
//
// Internal VPN package still uses role=relay for the agent; UI/fleet say secondary.
func ProvisionSecondary(o ProvisionSecondaryOpts) error {
	if o.Host == "" || o.Password == "" {
		return fmt.Errorf("host and password required")
	}
	if o.User == "" {
		o.User = "root"
	}
	if o.Port <= 0 {
		o.Port = 22
	}

	args := []string{"relay", "provision",
		"--host", o.Host,
		"--user", o.User,
		"--password", o.Password,
		"--port", fmt.Sprintf("%d", o.Port),
	}
	if strings.TrimSpace(o.SNI) != "" {
		args = append(args, "--sni", o.SNI)
	}
	fmt.Fprintln(os.Stderr, "==> secondary: VPN plane (relay provision)")
	cmd := exec.Command("netductor", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("relay provision: %w", err)
	}

	// Wait a moment for registry heartbeat
	time.Sleep(3 * time.Second)

	fmt.Fprintln(os.Stderr, "==> secondary: fleet roles + hostnames")
	_ = ensurePrimaryLocal()
	secID := findNodeIDByIP(o.Host)
	if secID != "" {
		_ = SetSecondary(secID)
		_, _ = nodes.SetDesiredHostname(secID, "nd-secondary")
	} else {
		fmt.Fprintln(os.Stderr, "  warn: secondary node id not in registry yet — run fleet bootstrap later")
	}

	fmt.Fprintln(os.Stderr, "==> secondary: data sync")
	_ = SyncPaths("root@" + o.Host)

	if !o.NoLampac {
		fmt.Fprintln(os.Stderr, "==> secondary: apply lampac on preferred node")
		if err := ApplyLampac(); err != nil {
			fmt.Fprintln(os.Stderr, "  lampac:", err)
		}
	}

	if !o.NoBotStandby {
		fmt.Fprintln(os.Stderr, "==> secondary: bot standby (SOCKS via primary)")
		prim := LoadPolicy().PrimarySSH
		if prim == "" {
			prim = "root@" + publicIPGuess()
		}
		// install units remotely
		script := fmt.Sprintf(`set -e
mkdir -p /opt/netductor/bin /etc/netductor/secrets
# binary already from relay join; ensure tg binary if present on primary will be scp'd by caller
if command -v netductor >/dev/null; then
  netductor fleet bot-standby-install %s || true
  netductor fleet bot-failover timer || true
fi
`, prim)
		c := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new",
			"root@"+o.Host, "bash", "-s")
		c.Stdin = strings.NewReader(script)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		_ = c.Run()
	}

	_ = InstallSyncTimer()
	fmt.Fprintln(os.Stderr, "==> secondary: done")
	fmt.Print(StatusSummary())
	return nil
}

func ensurePrimaryLocal() error {
	list, err := nodes.List()
	if err != nil {
		return err
	}
	localIP := publicIPGuess()
	for _, n := range list {
		if n.Kind == "vps" && (n.Role != "secondary" && n.Role != "relay") && (n.PublicIP == localIP || n.Role == "core") {
			_ = SetPrimary(n.ID)
			_, _ = nodes.SetDesiredHostname(n.ID, "nd-primary")
			return nil
		}
	}
	// fallback first non-relay
	for _, n := range list {
		if (n.Role != "secondary" && n.Role != "relay") {
			_ = SetPrimary(n.ID)
			_, _ = nodes.SetDesiredHostname(n.ID, "nd-primary")
			return nil
		}
	}
	return nil
}

func findNodeIDByIP(ip string) string {
	list, err := nodes.List()
	if err != nil {
		return ""
	}
	for _, n := range list {
		if n.PublicIP == ip {
			return n.ID
		}
	}
	// also match relay devices registered as nodes with role relay
	for _, n := range list {
		if (n.Role == "secondary" || n.Role == "relay") && (n.PublicIP == ip || strings.Contains(n.Hostname, "secondary") || strings.Contains(n.Hostname, "relay")) {
			if n.PublicIP == ip || n.PublicIP == "" {
				return n.ID
			}
		}
	}
	return ""
}
