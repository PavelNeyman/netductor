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
	MTLSCA   []byte
	MTLSCert []byte
	MTLSKey  []byte
	// ForceReinstall: explicit host-key rotation — forget Go TOFU for this host before dial.
	// Password alone is NOT enough to wipe trust (R6).
	ForceReinstall bool
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

// RemoteJoin installs netductor binary + joins with bundle JSON.
// Binary is verified against GitHub release SHA256SUMS when available.
func RemoteJoin(client *ssh.Client, bundleJSON string) (string, error) {
	ver := version.Release
	b64 := base64.StdEncoding.EncodeToString([]byte(bundleJSON))
	script := fmt.Sprintf(`set -e
export DEBIAN_FRONTEND=noninteractive
VER=%s
TD=$(mktemp -d /tmp/nd-join.XXXXXX)
chmod 700 "$TD"
trap 'rm -rf "$TD"; rm -f /root/bundle.json' EXIT
arch=$(uname -m)
case "$arch" in
  x86_64) ASSET=netductor-linux-amd64;;
  aarch64|arm64) ASSET=netductor-linux-arm64;;
  *) echo "unsupported arch $arch (need amd64 or arm64)" >&2; exit 1;;
esac
BASE="https://github.com/PavelNeyman/netductor/releases/download/v${VER}"
dl() {
  if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$1" "$2"
  else wget -qO "$1" "$2"; fi
}
dl "$TD/$ASSET" "$BASE/$ASSET" || { echo "download $ASSET v$VER failed" >&2; exit 1; }
if dl "$TD/SHA256SUMS" "$BASE/SHA256SUMS"; then
  EXPECT=$(grep -E "[[:space:]]${ASSET}$" "$TD/SHA256SUMS" | awk '{print $1}' | head -1)
  if [ -z "$EXPECT" ]; then echo "SHA256SUMS missing $ASSET" >&2; exit 1; fi
  GOT=$(sha256sum "$TD/$ASSET" | awk '{print $1}')
  if [ "$GOT" != "$EXPECT" ]; then echo "SHA256 mismatch: got $GOT want $EXPECT" >&2; exit 1; fi
else
  echo "SHA256SUMS download failed — refuse install without checksum" >&2
  exit 1
fi
# Prev binary outside $TD so trap does not erase rollback source.
PREV=
mkdir -p /var/lib/netductor
if [ -x /usr/local/bin/netductor ]; then
  cp -a /usr/local/bin/netductor /var/lib/netductor/netductor.bin.prev
  PREV=/var/lib/netductor/netductor.bin.prev
fi
rollback() {
  echo "join/health failed — restoring previous binary if any" >&2
  if [ -n "$PREV" ] && [ -f "$PREV" ]; then
    install -m 755 "$PREV" /usr/local/bin/netductor
    systemctl start netductor-secondary-agent 2>/dev/null || true
  fi
}
systemctl stop netductor-secondary-agent 2>/dev/null || true
install -m 755 "$TD/$ASSET" /usr/local/bin/netductor
echo '%s' | base64 -d > /root/bundle.json
chmod 600 /root/bundle.json
if ! netductor secondary join /root/bundle.json; then
  rollback
  exit 1
fi
if ! test "$(systemctl is-active sing-box)" = "active"; then
  rollback
  exit 1
fi
if ! test "$(systemctl is-active netductor-secondary-agent)" = "active"; then
  rollback
  exit 1
fi
if ! ss -tlnp | grep -qE ':443([^0-9]|$)'; then
  rollback
  exit 1
fi
rm -f /var/lib/netductor/netductor.bin.prev
`, ver, b64)
	return runSSH(client, script)
}

func installMTLSMaterial(client *ssh.Client, ca, cert, key []byte) error {
	if len(ca) == 0 || len(cert) == 0 || len(key) == 0 {
		return fmt.Errorf("mTLS material incomplete (ca=%d cert=%d key=%d bytes)", len(ca), len(cert), len(key))
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
	if len(in.MTLSCA) == 0 || len(in.MTLSCert) == 0 || len(in.MTLSKey) == 0 {
		return nil, fmt.Errorf("mTLS material required before secondary provision (ca=%d cert=%d key=%d)", len(in.MTLSCA), len(in.MTLSCert), len(in.MTLSKey))
	}
	// R6: never wipe TOFU on password alone. Only explicit ForceReinstall (or CLI ssh-hosts forget).
	if in.ForceReinstall {
		_ = ForgetSSHHost(in.Host)
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
		return nil, fmt.Errorf("mtls material: %w", err)
	}
	log.WriteString("mtls client material installed\n")
	out, err := RemoteJoin(client, bundleJSON)
	log.WriteString(out)
	pubJoined := strings.Join(keys, "\n")
	if err != nil {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String()}, fmt.Errorf("join: %w\n%s", err, out)
	}
	// Strict post-join checks (RemoteJoin already tests units; re-verify)
	sbOut, sbErr := runSSH(client, `test "$(systemctl is-active sing-box)" = "active"`)
	agOut, agErr := runSSH(client, `test "$(systemctl is-active netductor-secondary-agent)" = "active"`)
	_, p443 := runSSH(client, `ss -tlnp | grep -qE ':443([^0-9]|$)'`)
	agentOK := agErr == nil
	singOK := sbErr == nil && p443 == nil
	if !agentOK || !singOK {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: agentOK, SingBoxOK: singOK},
			fmt.Errorf("post-join services not healthy: agent_ok=%v singbox_ok=%v sb=%s ag=%s", agentOK, singOK, strings.TrimSpace(sbOut), strings.TrimSpace(agOut))
	}
	if err := DisablePasswordAuth(client); err != nil {
		log.WriteString("disable password: " + err.Error() + "\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			fmt.Errorf("join ok but disable password failed: %w", err)
	}
	log.WriteString("password auth disabled (harden-last)\n")
	return &ProvisionResult{
		Host: in.Host, PublicKey: pubJoined, Log: log.String(),
		AgentOK: true, SingBoxOK: true,
	}, nil
}
