package edge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

// parseSSHTargetHost returns host from "user@host" or "host".
func parseSSHTargetHost(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if i := strings.LastIndex(target, "@"); i >= 0 {
		return strings.TrimSpace(target[i+1:])
	}
	return target
}

// ensureEdgeHostKeyPin ensures a TOFU pin exists for the router (factory first contact
// auto-pins after ProbeHostKey — same intent as OpenSSH accept-new on day-0 LAN).
// Returns OpenSSH -o args for StrictHostKeyChecking=yes against the pin file.
func ensureEdgeHostKeyPin(sshTarget string) (hostKeyArgs []string, err error) {
	host := parseSSHTargetHost(sshTarget)
	if host == "" {
		return nil, fmt.Errorf("empty SSH host")
	}
	port := "22"
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".cache", "netductor", "ssh-pins")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	safe := strings.ReplaceAll(host, "/", "_")
	dest := filepath.Join(dir, safe+"_"+port+".known_hosts")

	write := func() error {
		return secondary.WriteOpenSSHKnownHostsFile(host, port, dest)
	}
	repin := func() error {
		key, perr := secondary.ProbeHostKey(host, port)
		if perr != nil {
			return fmt.Errorf("probe router host key %s:%s: %w", host, port, perr)
		}
		_ = secondary.ForgetSSHHost(host)
		_ = secondary.ForgetSSHHost(host + "|" + port)
		_ = os.Remove(dest)
		if terr := secondary.TrustHostKey(host, port, key); terr != nil {
			return fmt.Errorf("trust router host key %s:%s: %w", host, port, terr)
		}
		if err := write(); err != nil {
			return fmt.Errorf("materialize known_hosts after trust: %w", err)
		}
		return nil
	}
	if err := write(); err != nil {
		msg := err.Error()
		if errors.Is(err, secondary.ErrPinLost) || strings.Contains(msg, "pin lost") {
			// Edge reflash after pin-lost: allow re-pin on LAN (port 22).
			if err := repin(); err != nil {
				return nil, fmt.Errorf("SSH host key pin lost for %s:%s — recovery failed: %w", host, port, err)
			}
		} else if strings.Contains(msg, "no TOFU pin") {
			if err := repin(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		// Pin present: if live host key differs (factory reset), replace pin for edge LAN.
		live, perr := secondary.ProbeHostKey(host, port)
		if perr == nil {
			if terr := secondary.TrustHostKey(host, port, live); terr != nil && strings.Contains(terr.Error(), "host key mismatch") {
				fmt.Fprintf(os.Stderr, "==> edge SSH host key changed for %s — re-pin\n", host)
				if err := repin(); err != nil {
					return nil, err
				}
			}
		}
	}
	return []string{
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + dest,
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "HashKnownHosts=no",
	}, nil
}

// sshBaseForProvision builds ssh -o list: TOFU pin + password or key auth.
func sshBaseForProvision(opts ProvisionOpts) ([]string, error) {
	pinArgs, err := ensureEdgeHostKeyPin(opts.SSHTarget)
	if err != nil {
		return nil, err
	}
	useKeyOnly := strings.TrimSpace(opts.Password) == "" && opts.SSHKey != "" &&
		opts.NewRootPassword == "" && os.Getenv("NETDUCTOR_EDGE_KEY_ONLY") == "1"
	if useKeyOnly {
		return append(pinArgs, "-o", "BatchMode=yes", "-i", opts.SSHKey), nil
	}
	// Factory / pre-harden: password auth (empty password allowed).
	return append(pinArgs,
		"-o", "PreferredAuthentications=password",
		"-o", "PubkeyAuthentication=no",
		"-o", "NumberOfPasswordPrompts=3",
	), nil
}

func hardenSSHBasePinned(opts ProvisionOpts) ([]string, error) {
	pinArgs, err := ensureEdgeHostKeyPin(opts.SSHTarget)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.Password) == "" && opts.SSHKey != "" {
		return append(pinArgs, "-o", "BatchMode=yes", "-i", opts.SSHKey), nil
	}
	return append(pinArgs,
		"-o", "PreferredAuthentications=password",
		"-o", "PubkeyAuthentication=no",
		"-o", "NumberOfPasswordPrompts=3",
	), nil
}
