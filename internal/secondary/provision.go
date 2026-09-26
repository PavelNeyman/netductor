package secondary

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/version"
	"golang.org/x/crypto/ssh"
)

// ProvisionIn is operator input for zero-touch RU secondary setup from core (or Mac via primary).
type ProvisionIn struct {
	Host           string // IP or hostname
	Port           int    // SSH port, default 22
	User           string // default root
	Password       string
	SSHPrivateKey  string // optional path; used when password auth disabled (re-provision)
	SNI            string // Reality SNI; empty → ResolveSecondarySNI / api.vk.me
	OperatorPubKey string // optional: Mac/operator pubkey (preferred). If set, only this is installed.
	// Optional mTLS client material (written over the same SSH session — primary need not re-SSH later).
	MTLSCA     []byte
	MTLSCert   []byte
	MTLSKey    []byte
}

// ProvisionResult summarizes remote setup.
type ProvisionResult struct {
	Host      string `json:"host"`
	PublicKey string `json:"public_key_installed"`
	AgentOK   bool   `json:"agent_ok"`
	SingBoxOK bool   `json:"singbox_ok"`
	Log       string `json:"log"`
}

// CoreSSHPublicKey returns the pubkey on core (fallback when OperatorPubKey is empty).
// Ongoing primary↔secondary traffic is agent/HTTP; this key is only for emergency SSH from primary.
func CoreSSHPublicKey() (string, error) {
	candidates := []string{
		"/root/.ssh/id_ed25519.pub",
		"/root/.ssh/id_rsa.pub",
		filepath.Join(os.Getenv("HOME"), ".ssh/id_ed25519.pub"),
		filepath.Join(os.Getenv("HOME"), ".ssh/id_rsa.pub"),
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			s := strings.TrimSpace(string(b))
			if s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("no SSH public key on core — create /root/.ssh/id_ed25519 or pass --operator-pubkey")
}

func resolveInstallPubKeys(in ProvisionIn) ([]string, error) {
	if s := strings.TrimSpace(in.OperatorPubKey); s != "" {
		return []string{s}, nil
	}
	core, err := CoreSSHPublicKey()
	if err != nil {
		return nil, err
	}
	return []string{core}, nil
}

func sshClient(in ProvisionIn) (*ssh.Client, error) {
	if in.Port <= 0 {
		in.Port = 22
	}
	if in.User == "" {
		in.User = "root"
	}
	var methods []ssh.AuthMethod
	if k := strings.TrimSpace(in.SSHPrivateKey); k != "" {
		key, err := os.ReadFile(k)
		if err != nil {
			return nil, fmt.Errorf("read SSH private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse SSH private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if in.Password != "" {
		methods = append(methods, ssh.Password(in.Password))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("password or SSH private key required")
	}
	cfg := &ssh.ClientConfig{
		User:            in.User,
		Auth:            methods,
		HostKeyCallback: provisionHostKey(in.Host),
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(in.Host, fmt.Sprintf("%d", in.Port))
	return ssh.Dial("tcp", addr, cfg)
}

func runSSH(client *ssh.Client, cmd string) (string, error) {
	s, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	out, err := s.CombinedOutput(cmd)
	return string(out), err
}



// InstallOperatorKeys writes authorized_keys only (password auth stays enabled).
func InstallOperatorKeys(client *ssh.Client, pubKeys ...string) error {
	var installLines strings.Builder
	for _, pk := range pubKeys {
		pk = strings.TrimSpace(pk)
		if pk == "" {
			continue
		}
		esc := strings.ReplaceAll(pk, "'", `'"'"'`)
		installLines.WriteString(fmt.Sprintf("grep -qxF '%s' /root/.ssh/authorized_keys || echo '%s' >> /root/.ssh/authorized_keys\n", esc, esc))
	}
	if installLines.Len() == 0 {
		return fmt.Errorf("no public keys to install")
	}
	script := "set -e\nmkdir -p /root/.ssh\nchmod 700 /root/.ssh\ntouch /root/.ssh/authorized_keys\nchmod 600 /root/.ssh/authorized_keys\n" + installLines.String()
	_, err := runSSH(client, script)
	return err
}

// DisablePasswordAuth is the last step of a successful provision (harden-last).
// Shared with primary via hardening.RemoteHardenScript / DropInConf (Port 52222 + key-only).
func DisablePasswordAuth(client *ssh.Client) error {
	_, err := runSSH(client, hardening.RemoteHardenScript())
	return err
}

// HardenSSH = InstallOperatorKeys + DisablePasswordAuth (legacy one-shot).
func HardenSSH(client *ssh.Client, pubKeys ...string) error {
	if err := InstallOperatorKeys(client, pubKeys...); err != nil {
		return err
	}
	return DisablePasswordAuth(client)
}

// RemoteJoin installs netductor binary + joins with bundle JSON.
func RemoteJoin(client *ssh.Client, bundleJSON string) (string, error) {
	escaped := strings.ReplaceAll(bundleJSON, "'", `'"'"'`)
	ver := version.Release
	script := fmt.Sprintf("set -e\n"+
		"export DEBIAN_FRONTEND=noninteractive\n"+
		"VER=%s\n"+
		"dl() {\n"+
		"  u=\"https://github.com/PavelNeyman/netductor/releases/download/v$1/netductor-linux-amd64\"\n"+
		"  if command -v curl >/dev/null 2>&1; then curl -fsSL -o /tmp/netductor.new \"$u\"\n"+
		"  else wget -qO /tmp/netductor.new \"$u\"; fi\n"+
		"}\n"+
		"dl \"$VER\" || { echo \"download netductor v$VER failed\" >&2; exit 1; }\n"+
		"systemctl stop netductor-secondary-agent 2>/dev/null || true\n"+
		"install -m 755 /tmp/netductor.new /usr/local/bin/netductor\n"+
		"rm -f /tmp/netductor.new\n"+
		"cat > /root/bundle.json << 'BUNDLE_EOF'\n"+
		"%s\n"+
		"BUNDLE_EOF\n"+
		"netductor secondary join /root/bundle.json\n"+
		"systemctl is-active sing-box || true\n"+
		"systemctl is-active netductor-secondary-agent || true\n"+
		"ss -tlnp | grep -E ':443|:4443' || true\n",
		ver, escaped)
	return runSSH(client, script)
}

func installMTLSMaterial(client *ssh.Client, ca, cert, key []byte) error {
	if len(ca) == 0 || len(cert) == 0 || len(key) == 0 {
		return nil
	}
	enc := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	script := fmt.Sprintf("set -e\n"+
		"mkdir -p /etc/netductor/secrets/mtls\n"+
		"chmod 700 /etc/netductor/secrets /etc/netductor/secrets/mtls\n"+
		"echo '%s' | base64 -d > /etc/netductor/secrets/mtls/ca.crt\n"+
		"echo '%s' | base64 -d > /etc/netductor/secrets/mtls/client.crt\n"+
		"echo '%s' | base64 -d > /etc/netductor/secrets/mtls/client.key\n"+
		"chmod 600 /etc/netductor/secrets/mtls/ca.crt /etc/netductor/secrets/mtls/client.crt /etc/netductor/secrets/mtls/client.key\n",
		enc(ca), enc(cert), enc(key))
	_, err := runSSH(client, script)
	return err
}

// ProvisionFromCore: keys + mTLS + join while password works, then disable password (harden-last).
func ProvisionFromCore(in ProvisionIn, bundleJSON string) (*ProvisionResult, error) {
	var log strings.Builder
	keys, err := resolveInstallPubKeys(in)
	if err != nil {
		return nil, err
	}
	client, err := sshClient(in)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	log.WriteString("ssh connected\n")
	if err := InstallOperatorKeys(client, keys...); err != nil {
		return nil, fmt.Errorf("install keys: %w", err)
	}
	log.WriteString("operator pubkey installed (password still on)\n")
	if err := installMTLSMaterial(client, in.MTLSCA, in.MTLSCert, in.MTLSKey); err != nil {
		log.WriteString("mtls material: " + err.Error() + "\n")
	} else if len(in.MTLSCert) > 0 {
		log.WriteString("mtls client material installed\n")
	}
	out, err := RemoteJoin(client, bundleJSON)
	log.WriteString(out)
	pubJoined := strings.Join(keys, "\n")
	if err != nil {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String()}, fmt.Errorf("join: %w\n%s", err, out)
	}
	if err := DisablePasswordAuth(client); err != nil {
		log.WriteString("disable password: " + err.Error() + "\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(),
			AgentOK: strings.Contains(out, "active"), SingBoxOK: strings.Contains(out, "443"),
		}, fmt.Errorf("join ok but disable password failed: %w", err)
	}
	log.WriteString("password auth disabled (harden-last)\n")
	return &ProvisionResult{
		Host: in.Host, PublicKey: pubJoined, Log: log.String(),
		AgentOK: strings.Contains(out, "active"), SingBoxOK: strings.Contains(out, "443"),
	}, nil
}
