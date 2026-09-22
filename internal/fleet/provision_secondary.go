package fleet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

// ProvisionSecondaryOpts operator input for RU VPN-entry deploy from primary.
type ProvisionSecondaryOpts struct {
	Host           string
	User           string
	Password       string
	SSHPrivateKey  string // operator private key path (re-provision when password off)
	Port           int
	SNI            string
	OperatorPubKey string // Mac/operator pubkey; preferred over core key
}

// ProvisionSecondary deploys secondary as VPN entry only:
//  1) secondary provision (sing-box + agent + SSH harden with operator/core pubkey)
//  2) fleet secondary role + desired hostname nd-secondary
//
// Ongoing control plane is agent HTTP to primary — no permanent primary→secondary SSH required.
func ProvisionSecondary(o ProvisionSecondaryOpts) error {
	if o.Host == "" || (o.Password == "" && o.SSHPrivateKey == "") {
		return fmt.Errorf("host and password or ssh key required")
	}
	if o.User == "" {
		o.User = "root"
	}
	if o.Port <= 0 {
		o.Port = 22
	}

	args := []string{"secondary", "provision",
		"--host", o.Host,
		"--user", o.User,
		"--password", o.Password,
		"--port", fmt.Sprintf("%d", o.Port),
	}
	if strings.TrimSpace(o.SNI) != "" {
		args = append(args, "--sni", o.SNI)
	}
	if strings.TrimSpace(o.OperatorPubKey) != "" {
		args = append(args, "--operator-pubkey", o.OperatorPubKey)
	}
	if strings.TrimSpace(o.SSHPrivateKey) != "" {
		args = append(args, "--ssh-key", o.SSHPrivateKey)
	}
	fmt.Fprintln(os.Stderr, "==> secondary: VPN plane (secondary provision)")
	cmd := exec.Command("netductor", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("secondary provision: %w", err)
	}

	time.Sleep(3 * time.Second)

	fmt.Fprintln(os.Stderr, "==> secondary: fleet role + hostname")
	_ = ensurePrimaryLocal()
	secID := findNodeIDByIP(o.Host)
	if secID != "" {
		_ = SetSecondary(secID)
		_, _ = nodes.SetDesiredHostname(secID, "nd-secondary")
	} else {
		fmt.Fprintln(os.Stderr, "  warn: secondary node id not in registry yet — run fleet bootstrap later")
	}

	fmt.Fprintln(os.Stderr, "==> secondary: VPN entry ready (agent polls primary over HTTP)")
	fmt.Fprintln(os.Stderr, "  force user push: netductor secondary sync")
	return nil
}

func ensurePrimaryLocal() error {
	list, err := nodes.List()
	if err != nil {
		return err
	}
	for _, n := range list {
		if n.Role == "core" || strings.Contains(n.Hostname, "primary") || strings.Contains(n.Hostname, "core") {
			_ = SetPrimary(n.ID)
			_, _ = nodes.SetDesiredHostname(n.ID, "nd-primary")
			return nil
		}
	}
	return nil
}

func findNodeIDByIP(ip string) string {
	ip = strings.TrimSpace(ip)
	list, err := nodes.List()
	if err == nil {
		for _, n := range list {
			if n.PublicIP == ip || strings.Contains(n.PublicIP, ip) {
				return n.ID
			}
		}
	}
	for _, d := range secondary.List() {
		if d.PublicIP == ip || (ip != "" && strings.Contains(d.PublicIP, ip)) {
			return d.ID
		}
	}
	return ""
}

func publicIPGuess() string {
	out, _ := exec.Command("hostname", "-I").Output()
	fields := strings.Fields(string(out))
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}
