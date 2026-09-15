package secondary

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ProvisionIn is operator input for zero-touch RU relay setup from core.
type ProvisionIn struct {
	Host     string // IP or hostname
	Port     int    // SSH port, default 22
	User     string // default root
	Password string
	SNI      string // Reality SNI on relay; empty → ResolveRelaySNI / api.vk.me
}

// ProvisionResult summarizes remote setup.
type ProvisionResult struct {
	Host       string `json:"host"`
	PublicKey  string `json:"public_key_installed"`
	AgentOK    bool   `json:"agent_ok"`
	SingBoxOK  bool   `json:"singbox_ok"`
	Log        string `json:"log"`
}

// CoreSSHPublicKey returns the pubkey core will install on relay (same key for operator SSH).
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
	return "", fmt.Errorf("no SSH public key on core — create /root/.ssh/id_ed25519")
}

func sshClient(in ProvisionIn) (*ssh.Client, error) {
	if in.Port <= 0 {
		in.Port = 22
	}
	if in.User == "" {
		in.User = "root"
	}
	cfg := &ssh.ClientConfig{
		User: in.User,
		Auth: []ssh.AuthMethod{ssh.Password(in.Password)},
		HostKeyCallback: provisionHostKey(in.Host), // TOFU under state/relay/ssh_known_hosts.json
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

// HardenSSH installs core pubkey and disables password auth on remote.
func HardenSSH(client *ssh.Client, pubKey string) error {
	script := fmt.Sprintf(`set -e
mkdir -p /root/.ssh
chmod 700 /root/.ssh
touch /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
grep -qxF '%s' /root/.ssh/authorized_keys || echo '%s' >> /root/.ssh/authorized_keys
# disable password auth
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'SSH_EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
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
`, pubKey, pubKey)
	_, err := runSSH(client, script)
	return err
}

// RemoteJoin installs netductor binary + joins with bundle JSON written to /root/bundle.json.
func RemoteJoin(client *ssh.Client, bundleJSON string) (string, error) {
	// write bundle via base64 to avoid shell quoting issues
	b64 := strings.ReplaceAll(bundleJSON, "'", `'"'"'`)
	script := fmt.Sprintf(`set -e
export DEBIAN_FRONTEND=noninteractive
wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
cat > /root/bundle.json << 'BUNDLE_EOF'
%s
BUNDLE_EOF
netductor relay join /root/bundle.json
systemctl is-active sing-box || true
systemctl is-active netductor-relay-agent || true
ss -tlnp | grep -E ':443|:4443' || true
`, b64)
	return runSSH(client, script)
}

// ProvisionFromCore connects with password, installs pubkey, hardens SSH, runs relay join.
func ProvisionFromCore(in ProvisionIn, bundleJSON string) (*ProvisionResult, error) {
	var log strings.Builder
	pub, err := CoreSSHPublicKey()
	if err != nil {
		return nil, err
	}
	client, err := sshClient(in)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	log.WriteString("ssh connected\n")
	if err := HardenSSH(client, pub); err != nil {
		log.WriteString("harden: " + err.Error() + "\n")
	} else {
		log.WriteString("ssh key installed, password auth disabled\n")
	}
	out, err := RemoteJoin(client, bundleJSON)
	log.WriteString(out)
	if err != nil {
		return &ProvisionResult{Host: in.Host, PublicKey: pub, Log: log.String()}, fmt.Errorf("join: %w\n%s", err, out)
	}
	res := &ProvisionResult{
		Host: in.Host, PublicKey: pub, Log: log.String(),
		AgentOK: strings.Contains(out, "active"), SingBoxOK: strings.Contains(out, "443"),
	}
	return res, nil
}
