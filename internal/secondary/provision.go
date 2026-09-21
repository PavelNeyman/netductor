package secondary

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/version"
	"golang.org/x/crypto/ssh"
)

// ProvisionIn is operator input for zero-touch RU secondary setup from core (or Mac via primary).
type ProvisionIn struct {
	Host           string // IP or hostname
	Port           int    // SSH port, default 22
	User           string // default root
	Password       string
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
	cfg := &ssh.ClientConfig{
		User:            in.User,
		Auth:            []ssh.AuthMethod{ssh.Password(in.Password)},
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

// HardenSSH installs pubkey(s) and disables password auth on remote Debian/Ubuntu.
func HardenSSH(client *ssh.Client, pubKeys ...string) error {
	var installLines strings.Builder
	for _, pk := range pubKeys {
		pk = strings.TrimSpace(pk)
		if pk == "" {
			continue
		}
		// escape single quotes for shell
		esc := strings.ReplaceAll(pk, "'", `'"'"'`)
		installLines.WriteString(fmt.Sprintf("grep -qxF '%s' /root/.ssh/authorized_keys || echo '%s' >> /root/.ssh/authorized_keys\n", esc, esc))
	}
	if installLines.Len() == 0 {
		return fmt.Errorf("no public keys to install")
	}
	script := `set -e
mkdir -p /root/.ssh
chmod 700 /root/.ssh
touch /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
` + installLines.String() + `
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'SSH_EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
PermitRootLogin prohibit-password
PubkeyAuthentication yes
SSH_EOF
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  case "$f" in *netductor*) continue ;; esac
  sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/gi' "$f" 2>/dev/null || true
done
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
  sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
fi
systemctl reload sshd 2>/dev/null || systemctl reload ssh 2>/dev/null || service ssh reload 2>/dev/null || true
`
	_, err := runSSH(client, script)
	return err
}

// RemoteJoin installs netductor binary + joins with bundle JSON.
func RemoteJoin(client *ssh.Client, bundleJSON string) (string, error) {
	b64 := strings.ReplaceAll(bundleJSON, "'", `'"'"'`)
	script := fmt.Sprintf(`set -e
export DEBIAN_FRONTEND=noninteractive
VER=%s
wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v${VER}/netductor-linux-amd64 \
  || curl -fsSL -o /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v${VER}/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
cat > /root/bundle.json << 'BUNDLE_EOF'
%s
BUNDLE_EOF
netductor secondary join /root/bundle.json
systemctl is-active sing-box || true
systemctl is-active netductor-secondary-agent || true
ss -tlnp | grep -E ':443|:4443' || true
`, version.Release, b64)
	return runSSH(client, script)
}


// installMTLSMaterial writes CA+client cert/key on remote over an open SSH session.
func installMTLSMaterial(client *ssh.Client, ca, cert, key []byte) error {
	if len(ca) == 0 || len(cert) == 0 || len(key) == 0 {
		return nil
	}
	enc := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	script := fmt.Sprintf(`set -e
mkdir -p /etc/netductor/secrets/mtls
chmod 700 /etc/netductor/secrets /etc/netductor/secrets/mtls
echo '%s' | base64 -d > /etc/netductor/secrets/mtls/ca.crt
echo '%s' | base64 -d > /etc/netductor/secrets/mtls/client.crt
echo '%s' | base64 -d > /etc/netductor/secrets/mtls/client.key
chmod 600 /etc/netductor/secrets/mtls/ca.crt /etc/netductor/secrets/mtls/client.crt /etc/netductor/secrets/mtls/client.key
`, enc(ca), enc(cert), enc(key))
	_, err := runSSH(client, script)
	return err
}

// ProvisionFromCore connects with password, installs operator (or core) pubkey, hardens SSH, runs join.
// After this, ongoing control is agent→primary HTTP; SSH is for operator (Mac) only.
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
	if err := HardenSSH(client, keys...); err != nil {
		log.WriteString("harden: " + err.Error() + "\n")
	} else {
		log.WriteString("ssh key(s) installed, password auth disabled\n")
	}
	if err := installMTLSMaterial(client, in.MTLSCA, in.MTLSCert, in.MTLSKey); err != nil {
		log.WriteString("mtls material: " + err.Error() + "\n")
	} else if len(in.MTLSCert) > 0 {
		log.WriteString("mtls client material installed under /etc/netductor/secrets/mtls\n")
	}
	out, err := RemoteJoin(client, bundleJSON)
	log.WriteString(out)
	pubJoined := strings.Join(keys, "\n")
	if err != nil {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String()}, fmt.Errorf("join: %w\n%s", err, out)
	}
	return &ProvisionResult{
		Host: in.Host, PublicKey: pubJoined, Log: log.String(),
		AgentOK: strings.Contains(out, "active"), SingBoxOK: strings.Contains(out, "443"),
	}, nil
}
