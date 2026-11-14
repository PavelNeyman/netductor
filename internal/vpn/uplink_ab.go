package vpn

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"strconv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ssh"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Uplink AB: operator-only experimental S→P underlays (PLAN-UPLINK-AB-PROFILE).

const (
	uplinkABEnv      = "NETDUCTOR_UPLINK_AB"
	uplinkABUsersEnv = "NETDUCTOR_UPLINK_AB_USERS"
	uplinkABBasePort = 9444
	primaryABPortWS     = 10443
	primaryABPortGRPC   = 10444
	primaryABPortAnyTLS = 10445
	primaryABPortWG     = 51820
	abSSHSystemUser     = "nd-ab-uplink"

	// Reverse (P→S initiator) — client still hits S; L3 reverse WG, TCP underlays via WG VIP.
	uplinkABReverseBasePort = 9544
	secondaryRevWGPort      = 51821
)

// UplinkABProfile is one experimental path id.
type UplinkABProfile struct {
	ID          string
	InboundTag  string
	OutboundTag string
	Port        int
	Fragment    string
	PrimaryPort int
	Reverse     bool // P-initiated underlay (via reverse WG)
}

// UplinkABConfig file + env + embedded in SecondaryBundle.
type UplinkABConfig struct {
	Enabled  bool     `json:"enabled"`
	Users    []string `json:"users"`
	Profiles []string `json:"profiles"`
	BasePort int      `json:"base_port"`

	WGPrimaryPrivateKey   string `json:"wg_primary_private_key,omitempty"`
	WGPrimaryPublicKey    string `json:"wg_primary_public_key,omitempty"`
	WGSecondaryPrivateKey string `json:"wg_secondary_private_key,omitempty"`
	WGSecondaryPublicKey  string `json:"wg_secondary_public_key,omitempty"`
	WGPrimaryAddress      string `json:"wg_primary_address,omitempty"`
	WGSecondaryAddress    string `json:"wg_secondary_address,omitempty"`
	WGListenPort          int    `json:"wg_listen_port,omitempty"`
	WGEndpoint            string `json:"wg_endpoint,omitempty"`
	WGMTU                 int    `json:"wg_mtu,omitempty"`

	// Legacy aliases
	WGPrivateKey    string `json:"wg_private_key,omitempty"`
	WGPeerPublicKey string `json:"wg_peer_public_key,omitempty"`
	WGLocalAddress  string `json:"wg_local_address,omitempty"`
	WGPeerAddress   string `json:"wg_peer_address,omitempty"`

	// Reverse matrix (operator-only, same allowlist). Client → S:9544+; underlay initiator Primary.
	ReverseEnabled  bool     `json:"reverse_enabled"`
	ReverseProfiles []string `json:"reverse_profiles,omitempty"`
	ReverseBasePort int      `json:"reverse_base_port,omitempty"`

	WGRevPrimaryPrivateKey   string `json:"wg_rev_primary_private_key,omitempty"`
	WGRevPrimaryPublicKey    string `json:"wg_rev_primary_public_key,omitempty"`
	WGRevSecondaryPrivateKey string `json:"wg_rev_secondary_private_key,omitempty"`
	WGRevSecondaryPublicKey  string `json:"wg_rev_secondary_public_key,omitempty"`
	WGRevPrimaryAddress      string `json:"wg_rev_primary_address,omitempty"`
	WGRevSecondaryAddress    string `json:"wg_rev_secondary_address,omitempty"`
	WGRevListenPort          int    `json:"wg_rev_listen_port,omitempty"` // listen on secondary
	WGRevEndpoint            string `json:"wg_rev_endpoint,omitempty"`    // host:port primary dials (S public)

	SSHUser           string `json:"ssh_user,omitempty"`
	SSHPrivateKeyPath string `json:"ssh_private_key_path,omitempty"`
	SSHPrivateKey     string `json:"ssh_private_key,omitempty"`
	SSHHost           string `json:"ssh_host,omitempty"`
	SSHPort           int    `json:"ssh_port,omitempty"`
}

func uplinkABConfigPath() string {
	return filepath.Join(paths.EtcDir(), "uplink_ab.json")
}

func defaultUplinkABConfig() UplinkABConfig {
	return UplinkABConfig{
		Profiles:           []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"},
		BasePort:           uplinkABBasePort,
		WGPrimaryAddress:   "10.87.90.1/30",
		WGSecondaryAddress: "10.87.90.2/30",
		WGListenPort:       primaryABPortWG,
		WGMTU:              1280,
		SSHPort:            22,
		SSHUser:            abSSHSystemUser,
	}
}

// LoadUplinkABConfig merges file + env (env wins for enabled/users).
func LoadUplinkABConfig() UplinkABConfig {
	cfg := defaultUplinkABConfig()
	if b, err := os.ReadFile(uplinkABConfigPath()); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	normalizeUplinkAB(&cfg)
	if v := strings.TrimSpace(os.Getenv(uplinkABEnv)); v != "" {
		cfg.Enabled = v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "on")
	}
	if v := strings.TrimSpace(os.Getenv(uplinkABUsersEnv)); v != "" {
		cfg.Users = nil
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.Users = append(cfg.Users, p)
			}
		}
	}
	return cfg
}

func normalizeUplinkAB(cfg *UplinkABConfig) {
	if cfg.WGSecondaryPrivateKey == "" && cfg.WGPrivateKey != "" {
		cfg.WGSecondaryPrivateKey = cfg.WGPrivateKey
	}
	if cfg.WGPrimaryPublicKey == "" && cfg.WGPeerPublicKey != "" {
		cfg.WGPrimaryPublicKey = cfg.WGPeerPublicKey
	}
	if cfg.WGSecondaryAddress == "" && cfg.WGLocalAddress != "" {
		cfg.WGSecondaryAddress = cfg.WGLocalAddress
	}
	if cfg.WGPrimaryAddress == "" && cfg.WGPeerAddress != "" {
		if !strings.Contains(cfg.WGPeerAddress, "/") {
			cfg.WGPrimaryAddress = cfg.WGPeerAddress + "/30"
		} else {
			cfg.WGPrimaryAddress = cfg.WGPeerAddress
		}
	}
	if cfg.BasePort < 1024 || cfg.BasePort > 60000 {
		cfg.BasePort = uplinkABBasePort
	}
	if cfg.WGListenPort <= 0 {
		cfg.WGListenPort = primaryABPortWG
	}
	if cfg.WGMTU <= 0 {
		cfg.WGMTU = 1280
	}
	if cfg.WGPrimaryAddress == "" {
		cfg.WGPrimaryAddress = "10.87.90.1/30"
	}
	if cfg.WGSecondaryAddress == "" {
		cfg.WGSecondaryAddress = "10.87.90.2/30"
	}
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"}
	}
	if cfg.SSHPort <= 0 {
		cfg.SSHPort = 22
	}
	if cfg.SSHUser == "" {
		cfg.SSHUser = abSSHSystemUser
	}
	if cfg.ReverseBasePort < 1024 || cfg.ReverseBasePort > 60000 {
		cfg.ReverseBasePort = uplinkABReverseBasePort
	}
	if cfg.WGRevListenPort <= 0 {
		cfg.WGRevListenPort = secondaryRevWGPort
	}
	if cfg.WGRevPrimaryAddress == "" {
		cfg.WGRevPrimaryAddress = "10.87.91.1/30"
	}
	if cfg.WGRevSecondaryAddress == "" {
		cfg.WGRevSecondaryAddress = "10.87.91.2/30"
	}
	if cfg.ReverseEnabled && len(cfg.ReverseProfiles) == 0 {
		cfg.ReverseProfiles = knownReverseUplinkOrder()
	}
	// Drop legacy fake reverse transports (rev-ws etc. were labels on one WG).
	if len(cfg.ReverseProfiles) > 0 {
		var keep []string
		for _, id := range cfg.ReverseProfiles {
			id = strings.ToLower(strings.TrimSpace(id))
			if id == revSteerWG || id == revSteerPS {
				keep = append(keep, id)
			}
		}
		if cfg.ReverseEnabled && len(keep) == 0 {
			keep = knownReverseUplinkOrder()
		}
		cfg.ReverseProfiles = keep
	}
}

func saveUplinkABConfig(cfg UplinkABConfig) error {
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := uplinkABConfigPath() + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, uplinkABConfigPath())
}

// ApplyUplinkABFromBundle writes AB config carried in secondary bundle (keys + flags).
func ApplyUplinkABFromBundle(cfg *UplinkABConfig) error {
	if cfg == nil {
		return nil
	}
	normalizeUplinkAB(cfg)
	return saveUplinkABConfig(*cfg)
}

func UplinkABEnabled() bool {
	return LoadUplinkABConfig().Enabled
}

// UplinkABUserAllowed: explicit list, or operator / sub_profile=both.
func UplinkABUserAllowed(name string) bool {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" || name == RelayUplinkName {
		return false
	}
	if len(cfg.Users) > 0 {
		for _, u := range cfg.Users {
			if strings.EqualFold(strings.TrimSpace(u), name) {
				return true
			}
		}
		return false
	}
	if strings.EqualFold(name, "operator") {
		return true
	}
	r, err := loadRegistry()
	if err != nil {
		return false
	}
	u := findUser(r, name)
	if u == nil {
		return false
	}
	return EffectiveSubProfile(u) == SubBoth
}

func knownUplinkABOrder() []string {
	return []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"}
}

func profileWants(cfg UplinkABConfig, id string) bool {
	for _, p := range cfg.Profiles {
		if strings.EqualFold(strings.TrimSpace(p), id) {
			return true
		}
	}
	return false
}

func isLikelyPrimaryHost() bool {
	// Secondary agent unit or role marker.
	if _, err := os.Stat("/etc/systemd/system/netductor-agent.service"); err == nil {
		return false
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "role")); err == nil {
		return strings.TrimSpace(string(b)) == "primary"
	}
	return true
}

// EnsureUplinkABMaterial generates WG + SSH material and persists config.
func EnsureUplinkABMaterial() error {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled {
		return nil
	}
	changed := false

	if profileWants(cfg, "wg") {
		if cfg.WGPrimaryPrivateKey == "" || cfg.WGPrimaryPublicKey == "" {
			priv, pub, err := generateWGKeypair()
			if err != nil {
				return err
			}
			cfg.WGPrimaryPrivateKey = priv
			cfg.WGPrimaryPublicKey = pub
			changed = true
		}
		if cfg.WGSecondaryPrivateKey == "" || cfg.WGSecondaryPublicKey == "" {
			priv, pub, err := generateWGKeypair()
			if err != nil {
				return err
			}
			cfg.WGSecondaryPrivateKey = priv
			cfg.WGSecondaryPublicKey = pub
			changed = true
		}
		if cfg.WGEndpoint == "" {
			if ip := publicIP(); ip != "" {
				cfg.WGEndpoint = fmt.Sprintf("%s:%d", ip, cfg.WGListenPort)
				changed = true
			}
		}
	}

	if reverseProfileWants(cfg, "wg") || (cfg.ReverseEnabled && len(cfg.ReverseProfiles) > 0) {
		if cfg.WGRevPrimaryPrivateKey == "" || cfg.WGRevPrimaryPublicKey == "" {
			priv, pub, err := generateWGKeypair()
			if err != nil {
				return err
			}
			cfg.WGRevPrimaryPrivateKey = priv
			cfg.WGRevPrimaryPublicKey = pub
			changed = true
		}
		if cfg.WGRevSecondaryPrivateKey == "" || cfg.WGRevSecondaryPublicKey == "" {
			priv, pub, err := generateWGKeypair()
			if err != nil {
				return err
			}
			cfg.WGRevSecondaryPrivateKey = priv
			cfg.WGRevSecondaryPublicKey = pub
			changed = true
		}
		if cfg.WGRevEndpoint == "" && isLikelyPrimaryHost() {
			// Prefer online secondary public IP for P→S dial.
			for _, sec := range loadOnlineSecondarys() {
				ip := strings.TrimSpace(sec.PublicIP)
				if ip == "" {
					ip = strings.TrimSpace(sec.Host)
				}
				if ip != "" && net.ParseIP(ip) != nil {
					cfg.WGRevEndpoint = fmt.Sprintf("%s:%d", ip, cfg.WGRevListenPort)
					changed = true
					break
				}
			}
		}
	}

	if profileWants(cfg, "ssh") {
		keyPath := filepath.Join(paths.EtcDir(), "secrets", "uplink_ab_ssh")
		if cfg.SSHPrivateKeyPath == "" {
			cfg.SSHPrivateKeyPath = keyPath
			changed = true
		}
		if cfg.SSHHost == "" {
			if ip := publicIP(); ip != "" {
				cfg.SSHHost = ip
				changed = true
			}
		}
		if cfg.SSHUser == "" {
			cfg.SSHUser = abSSHSystemUser
			changed = true
		}
		if err := ensureABSSHKeypair(cfg.SSHPrivateKeyPath); err != nil {
			return err
		}
		if raw, err := os.ReadFile(cfg.SSHPrivateKeyPath); err == nil {
			if cfg.SSHPrivateKey != string(raw) {
				cfg.SSHPrivateKey = string(raw)
				changed = true
			}
		}
		if isLikelyPrimaryHost() {
			if err := ensureABSSHSystemUser(cfg.SSHPrivateKeyPath + ".pub"); err != nil {
				// non-fatal in tests / non-root
				fmt.Fprintln(os.Stderr, "uplink-ab ssh user:", err)
			}
		}
	}

	if changed {
		return saveUplinkABConfig(cfg)
	}
	return nil
}

func ensureABSSHKeypair(privPath string) error {
	if privPath == "" {
		return fmt.Errorf("empty ssh key path")
	}
	_ = os.MkdirAll(filepath.Dir(privPath), 0o700)
	if st, err := os.Stat(privPath); err == nil && st.Size() > 0 {
		return nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return err
	}
	block, err := ssh.MarshalPrivateKey(priv, "nd-ab-uplink")
	if err != nil {
		return err
	}
	if err := os.WriteFile(privPath, pem.EncodeToMemory(block), 0o600); err != nil {
		return err
	}
	auth := ssh.MarshalAuthorizedKey(sshPub)
	return os.WriteFile(privPath+".pub", auth, 0o644)
}

func ensureABSSHSystemUser(pubPath string) error {
	pub, err := os.ReadFile(pubPath)
	if err != nil {
		return err
	}
	// useradd -r -m -s /usr/sbin/nologin
	if _, err := exec.LookPath("useradd"); err != nil {
		return err
	}
	_ = exec.Command("useradd", "-r", "-m", "-s", "/usr/sbin/nologin", abSSHSystemUser).Run()
	home := filepath.Join("/home", abSSHSystemUser)
	if st, err := os.Stat(filepath.Join("/var/lib", abSSHSystemUser)); err == nil && st.IsDir() {
		home = filepath.Join("/var/lib", abSSHSystemUser)
	}
	// Debian system users often /var/lib or /
	if _, err := os.Stat(home); err != nil {
		home = filepath.Join("/var/lib", abSSHSystemUser)
		_ = os.MkdirAll(home, 0o700)
	}
	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0o700)
	ak := filepath.Join(sshDir, "authorized_keys")
	_ = os.WriteFile(ak, pub, 0o600)
	_ = exec.Command("chown", "-R", abSSHSystemUser+":"+abSSHSystemUser, home).Run()
	return nil
}


func ensureABTLSCerts() (certPath, keyPath string, err error) {
	dir := filepath.Join(paths.EtcDir(), "secrets")
	_ = os.MkdirAll(dir, 0o700)
	certPath = filepath.Join(dir, "uplink_ab_tls.crt")
	keyPath = filepath.Join(dir, "uplink_ab_tls.key")
	if st, e := os.Stat(certPath); e == nil && st.Size() > 0 {
		if st2, e2 := os.Stat(keyPath); e2 == nil && st2.Size() > 0 {
			return certPath, keyPath, nil
		}
	}
	// openssl if present
	if _, e := exec.LookPath("openssl"); e == nil {
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-keyout", keyPath,
			"-out", certPath, "-days", "3650", "-nodes", "-subj", "/CN=nd-uplink-ab")
		if out, e := cmd.CombinedOutput(); e != nil {
			return "", "", fmt.Errorf("openssl: %w (%s)", e, strings.TrimSpace(string(out)))
		}
		_ = os.Chmod(keyPath, 0o600)
		return certPath, keyPath, nil
	}
	return "", "", fmt.Errorf("openssl not found for uplink AB TLS cert")
}


func splitHostPortDefault(ep string, defPort int) (host string, port int) {
	host = strings.TrimSpace(ep)
	port = defPort
	if host == "" {
		return "", defPort
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		host = h
		if n, e := strconv.Atoi(p); e == nil && n > 0 {
			port = n
		}
		return host, port
	}
	// bare host
	return host, defPort
}

func generateWGKeypair() (privB64, pubB64 string, err error) {
	var priv, pub [32]byte
	if _, err = rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	curve25519.ScalarBaseMult(&pub, &priv)
	return base64.StdEncoding.EncodeToString(priv[:]), base64.StdEncoding.EncodeToString(pub[:]), nil
}

// ResolveUplinkABProfiles — after Ensure, all wanted profiles should resolve.
func ResolveUplinkABProfiles(cfg UplinkABConfig) []UplinkABProfile {
	if !cfg.Enabled {
		return nil
	}
	want := map[string]bool{}
	for _, id := range cfg.Profiles {
		want[strings.ToLower(strings.TrimSpace(id))] = true
	}
	var out []UplinkABProfile
	port := cfg.BasePort
	for _, id := range knownUplinkABOrder() {
		if !want[id] {
			continue
		}
		switch id {
		case "wg":
			if cfg.WGSecondaryPrivateKey == "" || cfg.WGPrimaryPublicKey == "" || cfg.WGEndpoint == "" {
				continue
			}
		case "ssh":
			if cfg.SSHHost == "" || cfg.SSHUser == "" || cfg.SSHPrivateKeyPath == "" {
				continue
			}
		}
		p := UplinkABProfile{
			ID: id, InboundTag: "ab-in-" + id, OutboundTag: "uplink-ab-" + id,
			Port: port, Fragment: "nd-up-" + id,
		}
		switch id {
		case "nomux":
			p.PrimaryPort = 0
		case "ws":
			p.PrimaryPort = primaryABPortWS
		case "grpc":
			p.PrimaryPort = primaryABPortGRPC
		case "anytls":
			p.PrimaryPort = primaryABPortAnyTLS
		case "wg":
			p.PrimaryPort = cfg.WGListenPort
		case "ssh":
			p.PrimaryPort = cfg.SSHPort
			if p.PrimaryPort <= 0 {
				p.PrimaryPort = 22
			}
		}
		out = append(out, p)
		port++
	}
	return out
}

func StatusUplinkAB() map[string]any {
	_ = EnsureUplinkABMaterial()
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	ids := make([]string, 0, len(profs))
	for _, p := range profs {
		ids = append(ids, fmt.Sprintf("%s:s%d→p%d", p.ID, p.Port, p.PrimaryPort))
	}
	rids := []string{}
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		rids = append(rids, fmt.Sprintf("%s:s%d→vip/p%d", p.ID, p.Port, p.PrimaryPort))
	}
	return map[string]any{
		"enabled":          cfg.Enabled,
		"users":            cfg.Users,
		"profiles":         ids,
		"reverse_enabled":  cfg.ReverseEnabled,
		"reverse_profiles": rids,
		"base_port":        cfg.BasePort,
		"reverse_base_port": cfg.ReverseBasePort,
		"wg_ready":         cfg.WGPrimaryPrivateKey != "" && cfg.WGSecondaryPrivateKey != "",
		"wg_rev_ready":     reverseWGReady(cfg),
		"wg_rev_endpoint":  cfg.WGRevEndpoint,
		"ssh_ready":        cfg.SSHPrivateKeyPath != "" && cfg.SSHHost != "",
		"config":           uplinkABConfigPath(),
	}
}

func appendUplinkABInbounds(inbounds []any, b *SecondaryBundle, privKey, shortID string, allUsers []secondaryUser) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	var abUsers []secondaryUser
	for _, u := range allUsers {
		if UplinkABUserAllowed(u.Name) {
			abUsers = append(abUsers, u)
		}
	}
	if len(profs) == 0 || len(abUsers) == 0 {
		return appendUplinkABReverseInbounds(inbounds, b, privKey, shortID, allUsers)
	}
	for _, p := range profs {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": p.InboundTag, "listen": "::", "listen_port": p.Port,
			"users": abUsers,
			"tls": map[string]any{
				"enabled": true, "server_name": b.SecondarySNI,
				"reality": map[string]any{
					"enabled": true,
					"handshake": map[string]any{
						"server": b.SecondarySNI, "server_port": 443,
					},
					"private_key": privKey,
					"short_id":    []string{shortID},
				},
			},
		})
	}
	return appendUplinkABReverseInbounds(inbounds, b, privKey, shortID, allUsers)
}

func coreDialHost(b *SecondaryBundle) string {
	h := primarySPAddrForSecondary()
	if h == "" {
		h = strings.TrimSpace(b.CoreIP)
	}
	return h
}

func appendUplinkABOutbounds(outbounds []any, b *SecondaryBundle) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	if len(profs) == 0 {
		return appendUplinkABReverseOutbounds(outbounds, b)
	}
	uplinkTLS := buildSecondaryUplinkTLS(b)
	coreHost := coreDialHost(b)
	// For public underlays (ws/grpc/anytls/wg/ssh) dial CoreIP not SP VIP when available
	publicCore := strings.TrimSpace(b.CoreIP)
	if publicCore == "" {
		publicCore = coreHost
	}
	for _, p := range profs {
		var ob map[string]any
		switch p.ID {
		case "nomux":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": b.CoreVless,
				"uuid": b.UplinkUUID, "tls": uplinkTLS,
			}
		case "wg":
			// WireGuard is an endpoint in sing-box 1.11+ (not outbound). See secondaryUplinkABEndpoints.
			continue
		case "anytls":
			ob = map[string]any{
				"type": "anytls", "tag": p.OutboundTag,
				"server": publicCore, "server_port": primaryABPortAnyTLS,
				"password": b.UplinkUUID,
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI,
					"insecure": true, // AB self-signed
				},
			}
		case "ws":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": publicCore, "server_port": primaryABPortWS,
				"uuid": b.UplinkUUID, // no vision: transport is ws
				"transport": map[string]any{"type": "ws", "path": "/nd-up-ws"},
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI, "insecure": true,
				},
			}
		case "grpc":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": publicCore, "server_port": primaryABPortGRPC,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{"type": "grpc", "service_name": "nd-up-grpc"},
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI, "insecure": true,
				},
			}
		case "ssh":
			port := cfg.SSHPort
			if port <= 0 {
				port = 22
			}
			host := cfg.SSHHost
			if host == "" {
				host = publicCore
			}
			ob = map[string]any{
				"type": "ssh", "tag": p.OutboundTag,
				"server": host, "server_port": port,
				"user": cfg.SSHUser,
			}
			if cfg.SSHPrivateKey != "" {
				ob["private_key"] = cfg.SSHPrivateKey
			} else if cfg.SSHPrivateKeyPath != "" {
				ob["private_key_path"] = cfg.SSHPrivateKeyPath
			}
		default:
			continue
		}
		outbounds = append(outbounds, ob)
	}
	return outbounds
}

func uplinkABRouteRules() []any {
	profs := ResolveUplinkABProfiles(LoadUplinkABConfig())
	var rules []any
	for _, p := range profs {
		rules = append(rules, map[string]any{
			"inbound":  []string{p.InboundTag},
			"outbound": p.OutboundTag,
		})
	}
	rules = append(rules, uplinkABReverseRouteRules()...)
	return rules
}

func appendPrimaryUplinkABInbounds(inbounds []any, uplinkUUID, sniVal, priv, sid string) []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || uplinkUUID == "" {
		return inbounds
	}
	u := []map[string]any{{"name": RelayUplinkName, "uuid": uplinkUUID, "flow": ""}}
	// WS/gRPC/AnyTLS: standard TLS cert (Reality is TCP-oriented; unknown fields / handshake mismatch break apply).
	certPath, keyPath, certErr := ensureABTLSCerts()
	tlsCert := map[string]any{"enabled": true, "server_name": sniVal}
	if certErr == nil {
		tlsCert["certificate_path"] = certPath
		tlsCert["key_path"] = keyPath
	}
	if profileWants(cfg, "ws") && certErr == nil {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": "ab-vless-ws", "listen": "::", "listen_port": primaryABPortWS,
			"users": u, "transport": map[string]any{"type": "ws", "path": "/nd-up-ws"}, "tls": tlsCert,
		})
	}
	if profileWants(cfg, "grpc") && certErr == nil {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": "ab-vless-grpc", "listen": "::", "listen_port": primaryABPortGRPC,
			"users": u, "transport": map[string]any{"type": "grpc", "service_name": "nd-up-grpc"}, "tls": tlsCert,
		})
	}
	if profileWants(cfg, "anytls") && certErr == nil {
		inbounds = append(inbounds, map[string]any{
			"type": "anytls", "tag": "ab-anytls", "listen": "::", "listen_port": primaryABPortAnyTLS,
			"users": []map[string]any{{"name": RelayUplinkName, "password": uplinkUUID}},
			"tls":   tlsCert,
		})
	}
	return inbounds
}

func primaryUplinkABEndpoints() []any {
	cfg := LoadUplinkABConfig()
	var eps []any
	if cfg.Enabled && profileWants(cfg, "wg") && cfg.WGPrimaryPrivateKey != "" && cfg.WGSecondaryPublicKey != "" {
		peerIP := cfg.WGSecondaryAddress
		if i := strings.IndexByte(peerIP, '/'); i > 0 {
			peerIP = peerIP[:i] + "/32"
		} else if peerIP != "" && !strings.Contains(peerIP, "/") {
			peerIP = peerIP + "/32"
		}
		eps = append(eps, map[string]any{
			"type":        "wireguard",
			"tag":         "ab-wg",
			"system":      false,
			"name":        "nd-ab-wg",
			"private_key": cfg.WGPrimaryPrivateKey,
			"listen_port": cfg.WGListenPort,
			"address":     []string{cfg.WGPrimaryAddress},
			"mtu":         cfg.WGMTU,
			"peers": []any{
				map[string]any{
					"public_key":  cfg.WGSecondaryPublicKey,
					"allowed_ips": []string{peerIP},
				},
			},
		})
	}
	eps = append(eps, primaryUplinkABReverseEndpoints()...)
	return eps
}

// openUplinkABFirewall best-effort ufw rules (primary + secondary ports).
func openUplinkABFirewall() {
	if _, err := exec.LookPath("ufw"); err != nil {
		return
	}
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled {
		return
	}
	if isLikelyPrimaryHost() {
		for _, p := range []int{primaryABPortWS, primaryABPortGRPC, primaryABPortAnyTLS} {
			_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p), "comment", "nd-uplink-ab").Run()
		}
		if profileWants(cfg, "wg") {
			_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", cfg.WGListenPort), "comment", "nd-uplink-ab-wg").Run()
		}
		return
	}
	for _, p := range ResolveUplinkABProfiles(cfg) {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p.Port), "comment", "nd-uplink-ab-steer").Run()
	}
	openUplinkABReverseFirewall()
}


// UplinkABSubscriptionLinks returns extra VLESS URIs for allowlisted user.
func UplinkABSubscriptionLinks(name, uuid, relayIP, pbk, sid, sniName string) []string {
	if !UplinkABUserAllowed(name) {
		return nil
	}
	_ = EnsureUplinkABMaterial()
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	rev := UplinkABReverseSubscriptionLinks(name, uuid, relayIP, pbk, sid, sniName)
	if len(profs) == 0 {
		return rev
	}
	if sniName == "" {
		sniName = DefaultRealitySNI
	}
	host := strings.TrimSpace(os.Getenv("NETDUCTOR_VPN_HOST"))
	if host == "" {
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "vpn_hostname")); err == nil {
			host = strings.TrimSpace(string(b))
		}
	}
	if host == "" {
		host = relayIP
	}
	var links []string
	for _, p := range profs {
		links = append(links, fmt.Sprintf(
			"vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#%s",
			uuid, host, p.Port, sniName, pbk, sid, p.Fragment,
		))
	}
	links = append(links, rev...)
	return links
}

// SnapshotUplinkAB returns config pointer for SecondaryBundle (after Ensure).
func SnapshotUplinkAB() *UplinkABConfig {
	if err := EnsureUplinkABMaterial(); err != nil {
		fmt.Fprintln(os.Stderr, "uplink-ab ensure:", err)
	}
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled {
		return nil
	}
	c := cfg
	return &c
}


// EnableUplinkAB turns on the matrix, optional explicit users (else operator + sub_both).
func EnableUplinkAB(users ...string) error {
	cfg := LoadUplinkABConfig()
	cfg.Enabled = true
	if len(users) > 0 {
		cfg.Users = users
	}
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = knownUplinkABOrder()
	}
	cfg.ReverseEnabled = true
	cfg.ReverseProfiles = knownReverseUplinkOrder()
	if err := saveUplinkABConfig(cfg); err != nil {
		return err
	}
	if err := EnsureUplinkABMaterial(); err != nil {
		return err
	}
	openUplinkABFirewall()
	// refresh subscription bodies so exp URIs appear without extra CLI
	r, err := loadRegistry()
	if err == nil {
		for _, u := range r.Users {
			if !u.Enabled {
				continue
			}
			if UplinkABUserAllowed(u.Name) {
				_ = WriteSubscriptionFiles(u.Name)
			}
		}
	}
	return nil
}

// DisableUplinkAB turns off experimental underlays (next apply drops inbounds).
func DisableUplinkAB() error {
	cfg := LoadUplinkABConfig()
	cfg.Enabled = false
	return saveUplinkABConfig(cfg)
}


// secondaryUplinkABEndpoints — WG client as sing-box endpoint (1.11+).
// Peers use address+port (NOT "endpoint" — that field does not exist in sing-box JSON).
func secondaryUplinkABEndpoints() []any {
	cfg := LoadUplinkABConfig()
	var eps []any
	if cfg.Enabled && profileWants(cfg, "wg") && cfg.WGSecondaryPrivateKey != "" && cfg.WGPrimaryPublicKey != "" && cfg.WGEndpoint != "" {
		h, pt := splitHostPortDefault(cfg.WGEndpoint, cfg.WGListenPort)
		if pt <= 0 {
			pt = primaryABPortWG
		}
		if h != "" {
			eps = append(eps, map[string]any{
				"type":        "wireguard",
				"tag":         "uplink-ab-wg",
				"system":      false,
				"name":        "nd-ab-wg-s",
				"private_key": cfg.WGSecondaryPrivateKey,
				"address":     []string{cfg.WGSecondaryAddress},
				"mtu":         cfg.WGMTU,
				"peers": []any{
					map[string]any{
						"address":     h,
						"port":        pt,
						"public_key":  cfg.WGPrimaryPublicKey,
						"allowed_ips": []string{"0.0.0.0/0"},
					},
				},
			})
		}
	}
	eps = append(eps, secondaryUplinkABReverseEndpoints()...)
	return eps
}


