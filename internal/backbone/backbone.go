// Package backbone — optional WireGuard service plane primary↔secondary (design: docs/BACKBONE-WG.md).
package backbone

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	"golang.org/x/crypto/curve25519"
)

const (
	DefaultPort       = 51820
	PrimaryAddrCIDR   = "10.87.0.1/30"
	SecondaryAddrCIDR = "10.87.0.2/30"
	PrimaryIP         = "10.87.0.1"
	SecondaryIP       = "10.87.0.2"
	InterfaceName     = "nd-backbone"
)

func Dir() string {
	if v := os.Getenv("NETDUCTOR_BACKBONE_DIR"); v != "" {
		return v
	}
	return filepath.Join(paths.EtcDir(), "backbone")
}

func statePath() string { return filepath.Join(Dir(), "state.json") }

// State is persisted after init-primary / import-secondary.
type State struct {
	Version          int    `json:"version"`
	Role             string `json:"role"` // primary | secondary
	Port             int    `json:"port"`
	PrimaryPublicKey string `json:"primary_public_key"`
	PrimaryPrivate   string `json:"primary_private_key,omitempty"`
	SecondaryPublic  string `json:"secondary_public_key"`
	SecondaryPrivate string `json:"secondary_private_key,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"` // primary host:port for secondary peer
	CreatedAt        string `json:"created_at"`
	Enabled          bool   `json:"enabled"`
}

func LoadState() (*State, error) {
	b, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func saveState(s *State) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), append(raw, '\n'), 0o600)
}

func genKeyPair() (privB64, pubB64 string, err error) {
	var priv [32]byte
	if _, err = rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	// WireGuard clamp
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)
	return base64.StdEncoding.EncodeToString(priv[:]), base64.StdEncoding.EncodeToString(pub[:]), nil
}

// InitPrimary generates keys and state for the EU primary.
func InitPrimary(port int, publicEndpoint string) (*State, error) {
	if port <= 0 {
		port = DefaultPort
	}
	pPriv, pPub, err := genKeyPair()
	if err != nil {
		return nil, err
	}
	sPriv, sPub, err := genKeyPair()
	if err != nil {
		return nil, err
	}
	ep := strings.TrimSpace(publicEndpoint)
	if ep != "" && !strings.Contains(ep, ":") {
		ep = fmt.Sprintf("%s:%d", ep, port)
	}
	s := &State{
		Version:          1,
		Role:             "primary",
		Port:             port,
		PrimaryPublicKey: pPub,
		PrimaryPrivate:   pPriv,
		SecondaryPublic:  sPub,
		SecondaryPrivate: sPriv,
		Endpoint:         ep,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		Enabled:          true,
	}
	if err := saveState(s); err != nil {
		return nil, err
	}
	if err := writeConfFiles(s); err != nil {
		return nil, err
	}
	return s, nil
}

// InitSecondaryFromState writes secondary conf from a primary-exported state (keys included).
func InitSecondaryFromState(s *State, endpoint string) error {
	if s == nil {
		return fmt.Errorf("nil state")
	}
	s.Role = "secondary"
	if endpoint != "" {
		s.Endpoint = endpoint
		if !strings.Contains(s.Endpoint, ":") {
			s.Endpoint = fmt.Sprintf("%s:%d", s.Endpoint, s.Port)
		}
	}
	if s.Endpoint == "" {
		return fmt.Errorf("endpoint required (primary public IP:port)")
	}
	if s.Port <= 0 {
		s.Port = DefaultPort
	}
	s.Enabled = true
	if err := saveState(s); err != nil {
		return err
	}
	return writeConfFiles(s)
}

func writeConfFiles(s *State) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	primaryConf := fmt.Sprintf(`[Interface]
# netductor backbone — primary
PrivateKey = %s
Address = %s
ListenPort = %d

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, s.PrimaryPrivate, PrimaryAddrCIDR, s.Port, s.SecondaryPublic, SecondaryIP)

	secConf := fmt.Sprintf(`[Interface]
# netductor backbone — secondary
PrivateKey = %s
Address = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s/32
PersistentKeepalive = 25
`, s.SecondaryPrivate, SecondaryAddrCIDR, s.PrimaryPublicKey, s.Endpoint, PrimaryIP)

	if err := os.WriteFile(filepath.Join(Dir(), "wg-primary.conf"), []byte(primaryConf), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(Dir(), "wg-secondary.conf"), []byte(secConf), 0o600); err != nil {
		return err
	}
	// Active conf for this role
	active := primaryConf
	if s.Role == "secondary" {
		active = secConf
	}
	return os.WriteFile(filepath.Join(Dir(), InterfaceName+".conf"), []byte(active), 0o600)
}

// ExportForSecondary returns JSON safe to copy to secondary (includes secondary private key).
func ExportForSecondary() ([]byte, error) {
	s, err := LoadState()
	if err != nil {
		return nil, err
	}
	out := *s
	out.PrimaryPrivate = "" // secondary does not need primary private
	return json.MarshalIndent(&out, "", "  ")
}

// Apply brings up interface via wg-quick if available.
func Apply() (string, error) {
	s, err := LoadState()
	if err != nil {
		return "", err
	}
	conf := filepath.Join(Dir(), InterfaceName+".conf")
	if _, err := os.Stat(conf); err != nil {
		if err := writeConfFiles(s); err != nil {
			return "", err
		}
	}
	// install into /etc/wireguard for wg-quick
	dst := filepath.Join("/etc/wireguard", InterfaceName+".conf")
	_ = os.MkdirAll("/etc/wireguard", 0o700)
	data, err := os.ReadFile(conf)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return "", err
	}
	if _, err := exec.LookPath("wg-quick"); err != nil {
		return "", fmt.Errorf("wg-quick not found — install wireguard-tools, conf written to %s", dst)
	}
	_ = exec.Command("wg-quick", "down", InterfaceName).Run()
	out, err := exec.Command("wg-quick", "up", InterfaceName).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return msg, fmt.Errorf("wg-quick up: %w (%s)", err, msg)
	}
	// firewall hint: open UDP port on primary
	if s.Role == "primary" {
		_ = exec.Command("bash", "-c", fmt.Sprintf(
			`command -v ufw >/dev/null && ufw allow from any to any port %d proto udp comment netductor-backbone 2>/dev/null || true`, s.Port)).Run()
	}
	return msg, nil
}

// StatusReport is doctor / CLI friendly.
type StatusReport struct {
	Configured bool   `json:"configured"`
	Role       string `json:"role,omitempty"`
	Port       int    `json:"port,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	IfaceUp    bool   `json:"iface_up"`
	Handshake  string `json:"last_handshake,omitempty"`
	Detail     string `json:"detail,omitempty"`
	PeerPing   string `json:"peer_ping,omitempty"`
}

func Status() StatusReport {
	s, err := LoadState()
	if err != nil {
		return StatusReport{Configured: false, Detail: "not configured"}
	}
	r := StatusReport{
		Configured: true,
		Role:       s.Role,
		Port:       s.Port,
		Endpoint:   s.Endpoint,
	}
	iface, err := net.InterfaceByName(InterfaceName)
	if err == nil && iface != nil {
		r.IfaceUp = iface.Flags&net.FlagUp != 0
	}
	if out, err := exec.Command("wg", "show", InterfaceName).CombinedOutput(); err == nil {
		r.Detail = strings.TrimSpace(string(out))
		for _, line := range strings.Split(r.Detail, "\n") {
			if strings.Contains(line, "latest handshake") {
				r.Handshake = strings.TrimSpace(line)
			}
		}
	} else if r.Detail == "" {
		r.Detail = "wg show failed (interface down or wireguard-tools missing)"
	}
	target := PrimaryIP
	if s.Role == "primary" {
		target = SecondaryIP
	}
	if out, err := exec.Command("ping", "-c", "1", "-W", "2", target).CombinedOutput(); err == nil {
		r.PeerPing = "ok"
	} else {
		r.PeerPing = "fail: " + strings.TrimSpace(string(out))
	}
	return r
}

func Enabled() bool {
	s, err := LoadState()
	return err == nil && s != nil && s.Enabled
}
