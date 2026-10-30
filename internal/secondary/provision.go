package secondary

import (
	"archive/tar"
	"bytes"
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
	Host          string // IP or hostname
	Port          int    // SSH port, default 22
	User          string // default root
	Password      string
	SSHPrivateKey string // optional path; used when password auth disabled (re-provision)
	// SSHKeyPassphrase unlocks SSHPrivateKey. Not applied to a different key.
	SSHKeyPassphrase string
	SNI              string // Reality SNI; empty → ResolveSecondarySNI / api.vk.me
	OperatorPubKey   string // optional: Mac/operator pubkey (preferred). If set, only this is installed.
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

// sshSignerFromIn loads the operator private key; required for key-only transport.
func sshSignerFromIn(in ProvisionIn) (ssh.Signer, error) {
	k := strings.TrimSpace(in.SSHPrivateKey)
	if k == "" {
		return nil, fmt.Errorf("SSH private key path required for key-only transport")
	}
	key, err := os.ReadFile(k)
	if err != nil {
		return nil, fmt.Errorf("read SSH private key: %w", err)
	}
	var signer ssh.Signer
	if in.SSHKeyPassphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(in.SSHKeyPassphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		return nil, fmt.Errorf("parse SSH private key: %w (encrypted key needs its passphrase; a separate secondary key does not inherit the primary passphrase)", err)
	}
	return signer, nil
}

func normalizeSSHIn(in *ProvisionIn) {
	if in.Port <= 0 {
		in.Port = 22
	}
	if in.User == "" {
		in.User = "root"
	}
}

// sshClientBootstrap: first contact. Password-only when password set (no key in Auth → no silent key fallback).
func sshClientBootstrap(in ProvisionIn) (*ssh.Client, error) {
	normalizeSSHIn(&in)
	var methods []ssh.AuthMethod
	if in.Password != "" {
		methods = append(methods, ssh.Password(in.Password))
	} else {
		signer, err := sshSignerFromIn(in)
		if err != nil {
			return nil, fmt.Errorf("secondary-bootstrap-connect: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("secondary-bootstrap-connect: password or SSH private key required")
	}
	cfg := &ssh.ClientConfig{
		User:            in.User,
		Auth:            methods,
		HostKeyCallback: provisionHostKey(in.Host, in.Port),
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(in.Host, fmt.Sprintf("%d", in.Port))
	return ssh.Dial("tcp", addr, cfg)
}

// sshClientKeyOnly: PublicKeys only — never password.
func sshClientKeyOnly(in ProvisionIn) (*ssh.Client, error) {
	normalizeSSHIn(&in)
	signer, err := sshSignerFromIn(in)
	if err != nil {
		return nil, fmt.Errorf("secondary-key-only-verification: %w", err)
	}
	cfg := &ssh.ClientConfig{
		User:            in.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: provisionHostKey(in.Host, in.Port),
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(in.Host, fmt.Sprintf("%d", in.Port))
	return ssh.Dial("tcp", addr, cfg)
}

// sshClient routes: password present → bootstrap; else key-only (resume).
func sshClient(in ProvisionIn) (*ssh.Client, error) {
	if strings.TrimSpace(in.Password) != "" {
		return sshClientBootstrap(in)
	}
	return sshClientKeyOnly(in)
}

// verifyKeyOnly opens a fresh TCP+SSH session and runs true.
func verifyKeyOnly(in ProvisionIn) (*ssh.Client, error) {
	c, err := sshClientKeyOnly(in)
	if err != nil {
		return nil, fmt.Errorf("secondary-key-only-verification: dial: %w", err)
	}
	if _, err := runSSH(c, "true"); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("secondary-key-only-verification: remote true: %w", err)
	}
	return c, nil
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

func runSSHWithStdin(client *ssh.Client, cmd string, stdin []byte) (string, error) {
	s, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	s.Stdin = bytes.NewReader(stdin)
	var buf bytes.Buffer
	s.Stdout = &buf
	s.Stderr = &buf
	err = s.Run(cmd)
	return buf.String(), err
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
// Bundle is passed only via SSH stdin (never base64 in remote argv).
func RemoteJoin(client *ssh.Client, bundleJSON string) (string, error) {
	ver := version.Release
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
  echo "join/health failed — restoring previous state if possible" >&2
  systemctl stop netductor-secondary-agent 2>/dev/null || true
  if [ -n "$PREV" ] && [ -f "$PREV" ]; then
    if install -m 755 "$PREV" /usr/local/bin/netductor 2>/dev/null; then
      systemctl start netductor-secondary-agent 2>/dev/null || true
    else
      echo "rollback: failed to restore previous binary" >&2
    fi
  else
    systemctl stop sing-box 2>/dev/null || true
    systemctl disable netductor-secondary-agent 2>/dev/null || true
    rm -f /usr/local/bin/netductor
  fi
  rm -f /root/bundle.json
}
# Any failure after we begin replacing the binary runs rollback.
trap 'rollback; exit 1' ERR
systemctl stop netductor-secondary-agent 2>/dev/null || true
install -m 755 "$TD/$ASSET" /usr/local/bin/netductor
# Bundle only via SSH stdin
cat > /root/bundle.json
chmod 600 /root/bundle.json
netductor secondary join /root/bundle.json
test "$(systemctl is-active sing-box)" = "active"
test "$(systemctl is-active netductor-secondary-agent)" = "active"
ss -tlnp | grep -qE ':443([^0-9]|$)'
trap - ERR
rm -f /var/lib/netductor/netductor.bin.prev
`, ver)
	_, err := runSSHWithStdin(client, script, []byte(bundleJSON))
	if err != nil {
		return "", fmt.Errorf("remote join failed")
	}
	return "join ok\n", nil
}

func installMTLSMaterial(client *ssh.Client, ca, cert, key []byte) error {
	if len(ca) == 0 || len(cert) == 0 || len(key) == 0 {
		return fmt.Errorf("mTLS material incomplete (ca=%d cert=%d key=%d bytes)", len(ca), len(cert), len(key))
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, data := range map[string][]byte{
		"ca.crt": ca, "client.crt": cert, "client.key": key,
	} {
		hdr := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(data))}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	script := `set -e
mkdir -p /etc/netductor/secrets/mtls
chmod 700 /etc/netductor/secrets /etc/netductor/secrets/mtls
tar -x -C /etc/netductor/secrets/mtls
chmod 600 /etc/netductor/secrets/mtls/ca.crt /etc/netductor/secrets/mtls/client.crt /etc/netductor/secrets/mtls/client.key
`
	_, err := runSSHWithStdin(client, script, buf.Bytes())
	if err != nil {
		return fmt.Errorf("mtls material install failed")
	}
	return nil
}

// ProvisionFromCore: password bootstrap → install pubkey → independent key-only gate → mTLS/join/harden on key session.
func ProvisionFromCore(in ProvisionIn, bundleJSON string) (*ProvisionResult, error) {
	var log strings.Builder
	// R6: never wipe TOFU on password alone. Only explicit ForceReinstall (or CLI ssh-hosts forget).
	// Fail closed before any remote work if trust reset cannot be applied.
	if in.ForceReinstall {
		if err := ForgetSSHHost(in.Host); err != nil {
			return nil, fmt.Errorf("ForceReinstall: cannot reset SSH trust for %s: %w", in.Host, err)
		}
	}
	keys, err := resolveInstallPubKeys(in)
	if err != nil {
		return nil, err
	}
	if len(in.MTLSCA) == 0 || len(in.MTLSCert) == 0 || len(in.MTLSKey) == 0 {
		return nil, fmt.Errorf("mTLS material required before secondary provision (ca=%d cert=%d key=%d)", len(in.MTLSCA), len(in.MTLSCert), len(in.MTLSKey))
	}
	// Stage A: bootstrap (password if provided; key-only resume otherwise).
	bootstrap, err := sshClient(in)
	if err != nil {
		return nil, fmt.Errorf("secondary-bootstrap-connect: %w", err)
	}
	log.WriteString("secondary-bootstrap-connect: ok\n")
	if err := InstallOperatorKeys(bootstrap, keys...); err != nil {
		_ = bootstrap.Close()
		return nil, fmt.Errorf("secondary-install-public-key: %w", err)
	}
	log.WriteString("secondary-install-public-key: ok\n")
	_ = bootstrap.Close()

	// Stage C: independent key-only gate (requires operator private key path).
	if strings.TrimSpace(in.SSHPrivateKey) == "" {
		return nil, fmt.Errorf("secondary-key-only-verification: operator private key path required after pubkey install")
	}
	client, err := verifyKeyOnly(in)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	// Stage D: drop password from memory — no password fallback for remainder.
	in.Password = ""
	log.WriteString("secondary-key-only-verification: ok; transport=key\n")

	if err := installMTLSMaterial(client, in.MTLSCA, in.MTLSCert, in.MTLSKey); err != nil {
		return nil, fmt.Errorf("mtls material: %w", err)
	}
	log.WriteString("mtls client material installed\n")
	_, err = RemoteJoin(client, bundleJSON)
	log.WriteString("remote join completed\n")
	pubJoined := strings.Join(keys, "\n")
	if err != nil {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String()}, fmt.Errorf("secondary-install-components: join failed: %w", err)
	}
	// Strict post-join checks (RemoteJoin already tests units; re-verify)
	_, sbErr := runSSH(client, `test "$(systemctl is-active sing-box)" = "active"`)
	_, agErr := runSSH(client, `test "$(systemctl is-active netductor-secondary-agent)" = "active"`)
	_, p443 := runSSH(client, `ss -tlnp | grep -qE ':443([^0-9]|$)'`)
	agentOK := agErr == nil
	singOK := sbErr == nil && p443 == nil
	if !agentOK || !singOK {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: agentOK, SingBoxOK: singOK},
			fmt.Errorf("post-join services not healthy: agent_ok=%v singbox_ok=%v", agentOK, singOK)
	}
	if err := DisablePasswordAuth(client); err != nil {
		log.WriteString("disable password: " + err.Error() + "\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			fmt.Errorf("secondary-password-auth-disable: %w", err)
	}
	log.WriteString("password auth disabled (harden-last)\n")
	return &ProvisionResult{
		Host: in.Host, PublicKey: pubJoined, Log: log.String(),
		AgentOK: true, SingBoxOK: true,
	}, nil
}
