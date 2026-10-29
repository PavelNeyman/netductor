package secondary

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// FingerprintSHA256 returns OpenSSH-style SHA256 fingerprint of a host key.
func FingerprintSHA256(key ssh.PublicKey) string {
	return ssh.FingerprintSHA256(key)
}

// ProbeHostKey dials host:port, captures the server host key during handshake, then aborts.
// Material is obtained over the unverified channel — operator must confirm fingerprint
// out-of-band (VPS console / existing trust) before TrustHostKey. Probe is not authentication.
func ProbeHostKey(host, port string) (ssh.PublicKey, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("empty host")
	}
	if strings.TrimSpace(port) == "" {
		port = "22"
	}
	addr := net.JoinHostPort(host, port)
	var captured ssh.PublicKey
	cfg := &ssh.ClientConfig{
		User: "netductor-probe",
		Auth: []ssh.AuthMethod{ssh.Password("__probe_no_auth__")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			captured = key
			return fmt.Errorf("probe: host key captured")
		},
		Timeout: 15 * time.Second,
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	if client != nil {
		_ = client.Close()
	}
	if captured != nil {
		return captured, nil
	}
	if err != nil {
		return nil, fmt.Errorf("probe host key %s: %w", addr, err)
	}
	return nil, fmt.Errorf("probe host key %s: no key captured", addr)
}

// TrustHostKey stores a TOFU pin for host after explicit operator confirmation of fingerprint.
// Does not replace an existing different pin (use ForgetSSHHost only after documented recovery).
// host identity matches provisionHostKey (host without port in JSON store).
func TrustHostKey(host, port string, key ssh.PublicKey) error {
	if key == nil {
		return fmt.Errorf("nil host key")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	got := base64.StdEncoding.EncodeToString(key.Marshal())
	pkhMu.Lock()
	defer pkhMu.Unlock()
	f, err := loadRelayKH()
	if err != nil {
		return err
	}
	if err := ensureSeenMigratedFromPins(f); err != nil {
		return err
	}
	if prev, ok := f[host]; ok {
		if prev != got {
			return fmt.Errorf("host key mismatch for %s: existing pin differs — refuse replace (documented recovery: ssh-hosts forget)", host)
		}
		_ = markHostSeen(host)
		return nil
	}
	if wasSeen, err := loadSeenHosts(); err != nil {
		return err
	} else if wasSeen[host] {
		return ErrPinLost
	}
	f[host] = got
	if err := saveRelayKH(f); err != nil {
		return err
	}
	return markHostSeen(host)
}

// TrustHostKeyIfFingerprintMatch probes the server, compares SHA256 fingerprint, and pins on match.
// expectedFP must be the full OpenSSH form from FingerprintSHA256 (prefix "SHA256:").
func TrustHostKeyIfFingerprintMatch(host, port, expectedFP string) (ssh.PublicKey, error) {
	expectedFP = strings.TrimSpace(expectedFP)
	if expectedFP == "" {
		return nil, fmt.Errorf("fingerprint required (out-of-band confirmation)")
	}
	key, err := ProbeHostKey(host, port)
	if err != nil {
		return nil, err
	}
	got := FingerprintSHA256(key)
	if !strings.EqualFold(got, expectedFP) {
		return key, fmt.Errorf("fingerprint mismatch for %s:%s: got %s want %s", host, port, got, expectedFP)
	}
	if err := TrustHostKey(host, port, key); err != nil {
		return key, err
	}
	return key, nil
}

// HasTOFUPin reports whether host has a stored pin (for deploy preflight).
func HasTOFUPin(host string) (bool, error) {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	pkhMu.Lock()
	defer pkhMu.Unlock()
	f, err := loadRelayKH()
	if err != nil {
		return false, err
	}
	_, ok := f[host]
	return ok, nil
}
