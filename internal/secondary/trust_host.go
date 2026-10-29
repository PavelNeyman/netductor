package secondary

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// endpointID is the TOFU map key: "host|port" (port defaults to 22).
// Legacy pre-0.9.605 entries used bare "host"; lookupPin still accepts them.
func endpointID(host, port string) string {
	host = normalizeHost(host)
	port = strings.TrimSpace(port)
	if port == "" {
		port = "22"
	}
	return host + "|" + port
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return strings.Trim(host, "[]")
}

// lookupPin finds a pin for host:port. Prefers port-aware id, then legacy bare host.
func lookupPin(f map[string]string, host, port string) (id, b64 string, ok bool) {
	id = endpointID(host, port)
	if v := strings.TrimSpace(f[id]); v != "" {
		return id, v, true
	}
	h := normalizeHost(host)
	if v := strings.TrimSpace(f[h]); v != "" {
		return h, v, true
	}
	return id, "", false
}

func seenEndpoint(seen map[string]bool, host, port string) bool {
	if seen[endpointID(host, port)] {
		return true
	}
	return seen[normalizeHost(host)]
}

func hasAnyPortAwarePin(f map[string]string, host string) bool {
	host = normalizeHost(host)
	prefix := host + "|"
	for k, v := range f {
		if strings.HasPrefix(k, prefix) && strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

// FingerprintSHA256 returns OpenSSH-style SHA256 fingerprint of a host key.
func FingerprintSHA256(key ssh.PublicKey) string {
	return ssh.FingerprintSHA256(key)
}

// ProbeHostKey dials host:port, captures the server host key during handshake, then aborts.
// Material is obtained over the unverified channel — operator must confirm fingerprint
// out-of-band before TrustHostKey. Probe is not authentication.
func ProbeHostKey(host, port string) (ssh.PublicKey, error) {
	host = normalizeHost(host)
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

// TrustHostKey stores a TOFU pin for host:port after explicit operator confirmation.
// Does not replace an existing different pin. New entries use port-aware endpointID.
func TrustHostKey(host, port string, key ssh.PublicKey) error {
	if key == nil {
		return fmt.Errorf("nil host key")
	}
	host = normalizeHost(host)
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if strings.TrimSpace(port) == "" {
		port = "22"
	}
	got := base64.StdEncoding.EncodeToString(key.Marshal())
	id := endpointID(host, port)

	pkhMu.Lock()
	defer pkhMu.Unlock()
	f, err := loadRelayKH()
	if err != nil {
		return err
	}
	if err := ensureSeenMigratedFromPins(f); err != nil {
		return err
	}
	// Match existing port-aware or legacy host-only pin.
	if prevID, prev, ok := lookupPin(f, host, port); ok {
		if prev != got {
			return fmt.Errorf("host key mismatch for %s: existing pin differs — refuse replace (documented recovery: ssh-hosts forget)", prevID)
		}
		// Upgrade legacy bare-host entry to port-aware when same key.
		if prevID != id {
			f[id] = got
			if err := saveRelayKH(f); err != nil {
				return err
			}
		}
		return markHostSeen(id)
	}
	seen, err := loadSeenHosts()
	if err != nil {
		return err
	}
	// Pin-lost only when this endpoint (or legacy bare host with no port-aware pins) was seen.
	if seen[id] {
		return fmt.Errorf("%w: %s", ErrPinLost, id)
	}
	if seen[host] && !hasAnyPortAwarePin(f, host) {
		return fmt.Errorf("%w: %s", ErrPinLost, id)
	}
	f[id] = got
	if err := saveRelayKH(f); err != nil {
		return err
	}
	if err := markHostSeen(id); err != nil {
		return fmt.Errorf("pin saved but seen mark failed: %w", err)
	}
	return nil
}

// TrustHostKeyIfFingerprintMatch probes, compares fingerprint with exact string match, then pins.
// expectedFP must equal FingerprintSHA256 output (including "SHA256:" prefix); case-sensitive.
func TrustHostKeyIfFingerprintMatch(host, port, expectedFP string) (ssh.PublicKey, error) {
	expectedFP = strings.TrimSpace(expectedFP)
	if expectedFP == "" {
		return nil, fmt.Errorf("fingerprint required (out-of-band confirmation)")
	}
	key, err := ProbeHostKey(host, port)
	if err != nil {
		return nil, err
	}
	got := strings.TrimSpace(FingerprintSHA256(key))
	if got != expectedFP {
		return key, fmt.Errorf("fingerprint mismatch for %s:%s: got %s want %s", host, port, got, expectedFP)
	}
	if err := TrustHostKey(host, port, key); err != nil {
		return key, err
	}
	return key, nil
}

// PinStatus is the preflight classification for an endpoint.
type PinStatus int

const (
	PinAbsent  PinStatus = iota // never trusted
	PinPresent                  // pin available
	PinLost                     // previously known, pin missing
)

// CheckTOFUPin classifies host:port trust state for deploy preflight.
func CheckTOFUPin(host, port string) (PinStatus, error) {
	host = normalizeHost(host)
	if host == "" {
		return PinAbsent, fmt.Errorf("empty host")
	}
	if strings.TrimSpace(port) == "" {
		port = "22"
	}
	pkhMu.Lock()
	defer pkhMu.Unlock()
	f, err := loadRelayKH()
	if err != nil {
		return PinAbsent, err
	}
	if _, _, ok := lookupPin(f, host, port); ok {
		return PinPresent, nil
	}
	seen, err := loadSeenHosts()
	if err != nil {
		return PinAbsent, err
	}
	id := endpointID(host, port)
	if seen[id] {
		return PinLost, nil
	}
	// Legacy bare-host seen only if no remaining port-aware pins for this host.
	if seen[host] && !hasAnyPortAwarePin(f, host) {
		return PinLost, nil
	}
	return PinAbsent, nil
}

// HasTOFUPin reports whether a usable pin exists (legacy helper).
func HasTOFUPin(host string) (bool, error) {
	st, err := CheckTOFUPin(host, "52222")
	if err != nil {
		return false, err
	}
	return st == PinPresent, nil
}

// HasTOFUPinPort is port-aware.
func HasTOFUPinPort(host, port string) (bool, error) {
	st, err := CheckTOFUPin(host, port)
	if err != nil {
		return false, err
	}
	return st == PinPresent, nil
}
