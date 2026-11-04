package secondary

import (
	"archive/tar"
	"bytes"
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

func validateOperatorPubKey(pk string) (string, error) {
	pk = strings.TrimSpace(pk)
	if pk == "" || strings.ContainsAny(pk, "\r\n") {
		return "", fmt.Errorf("operator SSH public key must be exactly one non-empty line")
	}
	key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(pk))
	if err != nil || key == nil || len(options) != 0 || strings.TrimSpace(string(rest)) != "" {
		return "", fmt.Errorf("invalid operator SSH public key: expected one plain authorized-key entry")
	}
	return pk, nil
}

func resolveInstallPubKeys(in ProvisionIn) ([]string, error) {
	var candidate string
	if s := strings.TrimSpace(in.OperatorPubKey); s != "" {
		candidate = s
	} else {
		core, err := CoreSSHPublicKey()
		if err != nil {
			return nil, err
		}
		candidate = core
	}
	validated, err := validateOperatorPubKey(candidate)
	if err != nil {
		return nil, err
	}
	return []string{validated}, nil
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
	if in.Port > 65535 {
		return nil, fmt.Errorf("invalid SSH port %d: expected 1..65535", in.Port)
	}
	normalizeSSHIn(&in)
	var methods []ssh.AuthMethod
	if in.Password != "" {
		pw := in.Password
		methods = append(methods, ssh.Password(pw))
		// Many cloud images accept keyboard-interactive but not the "password" method;
		// OpenSSH CLI succeeds while ssh.Password alone fails with "no supported methods".
		methods = append(methods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
			answers = make([]string, len(questions))
			for i := range questions {
				answers[i] = pw
			}
			return answers, nil
		}))
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
	if in.Port > 65535 {
		return nil, fmt.Errorf("invalid SSH port %d: expected 1..65535", in.Port)
	}
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
		var err error
		pk, err = validateOperatorPubKey(pk)
		if err != nil {
			return fmt.Errorf("%s: %w", StageInstallPublicKey, err)
		}
		esc := strings.ReplaceAll(pk, "'", `'"'"'`)
		installLines.WriteString(fmt.Sprintf("grep -qxF '%s' /root/.ssh/authorized_keys || echo '%s' >> /root/.ssh/authorized_keys\n", esc, esc))
	}
	if installLines.Len() == 0 {
		return fmt.Errorf("no public keys to install")
	}
	script := "set -e\nmkdir -p /root/.ssh\nchmod 700 /root/.ssh\ntouch /root/.ssh/authorized_keys\nchmod 600 /root/.ssh/authorized_keys\n" + installLines.String()
	// Verify each key line is present after install.
	for _, pk := range pubKeys {
		pk = strings.TrimSpace(pk)
		if pk == "" {
			continue
		}
		esc := strings.ReplaceAll(pk, "'", `'"'"'`)
		script += fmt.Sprintf("grep -qxF '%s' /root/.ssh/authorized_keys\n", esc)
	}
	_, err := runSSH(client, script)
	if err != nil {
		return fmt.Errorf("%s: %w", StageInstallPublicKey, err)
	}
	return nil
}

// DisablePasswordAuth is disabled: one-shot would skip independent key-only verification.
// Use HardenSSHPrepare → verifyKeyOnly → HardenSSHFinalize → verifyKeyOnly → HardenSSHCancelRollback.
func DisablePasswordAuth(client *ssh.Client) error {
	return fmt.Errorf("DisablePasswordAuth removed: use phased HardenSSHPrepare/Finalize/CancelRollback with independent key-only checks")
}

// HardenSSHPrepare moves sshd to DefaultSSHPort, keeps :22 open, arms rollback timer.
func HardenSSHPrepare(client *ssh.Client) error {
	out, err := runSSH(client, hardening.RemoteHardenPrepareScript())
	if err != nil {
		return fmt.Errorf("%s: %w (%s)", StageSSHHardenPrepare, err, trimLog(out))
	}
	return nil
}

// HardenSSHFinalize disables password and denies :22. Does NOT cancel rollback timer.
func HardenSSHFinalize(client *ssh.Client) error {
	out, err := runSSH(client, hardening.RemoteHardenFinalizeScript())
	if err != nil {
		return fmt.Errorf("%s: %w (%s)", StageHardenFinalize, err, trimLog(out))
	}
	return nil
}

// HardenSSHCancelRollback stops the independent rollback timer after proven key-only access.
func HardenSSHCancelRollback(client *ssh.Client) error {
	out, err := runSSH(client, hardening.RemoteHardenCancelRollbackScript())
	if err != nil {
		return fmt.Errorf("%s: cancel rollback: %w (%s)", StageHardenFinalize, err, trimLog(out))
	}
	return nil
}

// HardenSSHHostAgentPurge removes hoster agents after SSH access is locked in (post-cancel).
func HardenSSHHostAgentPurge(client *ssh.Client) error {
	out, err := runSSH(client, hardening.RemoteHostAgentPurgeScript())
	if err != nil {
		return fmt.Errorf("secondary-host-agent-purge: %w (%s)", err, trimLog(out))
	}
	return nil
}

func trimLog(s string) string {
	s = strings.TrimSpace(s)
	const max = 4000
	if len(s) <= max {
		return s
	}
	// Keep the end: join/baseline failures append the real reason last.
	return "…" + s[len(s)-max:]
}

// Legacy HardenSSH one-shot removed; use phased prepare/finalize/cancel only.

// RemoteJoin installs netductor binary + joins with bundle JSON.
// Binary is verified against GitHub release SHA256SUMS when available.
// Bundle is passed only via SSH stdin (never base64 in remote argv).
func RemoteJoin(client *ssh.Client, bundleJSON string) (string, error) {
	ver := version.Release
	if strings.TrimSpace(bundleJSON) == "" {
		return "", fmt.Errorf("remote join: empty bundle")
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(bundleJSON))
	// Bundle as base64 (not session stdin). Progress lines so failures are never silent.
	script := "set -eu\n" +
		"export DEBIAN_FRONTEND=noninteractive\n" +
		"VER=" + ver + "\n" +
		"echo join:start ver=$VER arch=$(uname -m)\n" +
		"TD=$(mktemp -d /tmp/nd-join.XXXXXX)\n" +
		"chmod 700 \"$TD\"\n" +
		"NEW=; PREV_TMP=; RESTORE=; PREV=\n" +
		"nd_sync() { sync -f \"$1\" 2>/dev/null || sync 2>/dev/null || true; }\n" +
		"trap 'rm -rf \"$TD\"; rm -f /root/bundle.json \"${NEW:-}\" \"${PREV_TMP:-}\" \"${RESTORE:-}\"' EXIT\n" +
		"NEW=$(mktemp /usr/local/bin/.netductor.new.XXXXXX)\n" +
		"arch=$(uname -m)\n" +
		"case \"$arch\" in\n" +
		"  x86_64) ASSET=netductor-linux-amd64;;\n" +
		"  aarch64|arm64) ASSET=netductor-linux-arm64;;\n" +
		"  *) echo \"unsupported arch $arch\" >&2; exit 1;;\n" +
		"esac\n" +
		"BASE=\"https://github.com/PavelNeyman/netductor/releases/download/v${VER}\"\n" +
		"dl() {\n" +
		"  i=1\n" +
		"  while [ \"$i\" -le 4 ]; do\n" +
		"    if command -v curl >/dev/null 2>&1; then\n" +
		"      curl -fsSL --http1.1 --retry 2 --retry-all-errors --proto \"=https\" --proto-redir \"=https\" -o \"$1\" \"$2\" && return 0\n" +
		"    else\n" +
		"      wget --https-only -qO \"$1\" \"$2\" && return 0\n" +
		"    fi\n" +
		"    echo \"download retry $i: $2\" >&2\n" +
		"    i=$((i+1))\n" +
		"    sleep 2\n" +
		"  done\n" +
		"  return 1\n" +
		"}\n" +
		"echo join:download $ASSET\n" +
		"dl \"$TD/$ASSET\" \"$BASE/$ASSET\" || { echo \"download $ASSET v$VER failed\" >&2; exit 1; }\n" +
		"echo join:download SHA256SUMS\n" +
		"dl \"$TD/SHA256SUMS\" \"$BASE/SHA256SUMS\" || { echo \"SHA256SUMS download failed\" >&2; exit 1; }\n" +
		"EXPECT=$(grep -E \"[[:space:]]${ASSET}$\" \"$TD/SHA256SUMS\" | awk '{print $1}' | head -1)\n" +
		"[ -n \"$EXPECT\" ] || { echo \"SHA256SUMS missing $ASSET\" >&2; exit 1; }\n" +
		"GOT=$(sha256sum \"$TD/$ASSET\" | awk '{print $1}')\n" +
		"[ \"$GOT\" = \"$EXPECT\" ] || { echo \"SHA256 mismatch got=$GOT want=$EXPECT\" >&2; exit 1; }\n" +
		"echo join:checksum_ok\n" +
		"mkdir -p /var/lib/netductor\n" +
		"if [ -e /usr/local/bin/netductor ]; then\n" +
		"  test -f /usr/local/bin/netductor && test ! -L /usr/local/bin/netductor || { echo \"existing netductor not a regular file\" >&2; exit 1; }\n" +
		"  PREV_TMP=$(mktemp /var/lib/netductor/.netductor.bin.prev.XXXXXX)\n" +
		"  cp -a /usr/local/bin/netductor \"$PREV_TMP\"\n" +
		"  nd_sync \"$PREV_TMP\"\n" +
		"  mv -f \"$PREV_TMP\" /var/lib/netductor/netductor.bin.prev\n" +
		"  PREV_TMP=\n" +
		"  nd_sync /var/lib/netductor\n" +
		"  PREV=/var/lib/netductor/netductor.bin.prev\n" +
		"fi\n" +
		"rollback() {\n" +
		"  echo \"join/health failed — rollback\" >&2\n" +
		"  systemctl stop netductor-secondary-agent 2>/dev/null || true\n" +
		"  systemctl stop sing-box 2>/dev/null || true\n" +
		"  if [ -n \"${PREV:-}\" ] && [ -f \"$PREV\" ]; then\n" +
		"    RESTORE=$(mktemp /usr/local/bin/.netductor.restore.XXXXXX)\n" +
		"    if install -m 755 \"$PREV\" \"$RESTORE\" 2>/dev/null && nd_sync \"$RESTORE\" && mv -f \"$RESTORE\" /usr/local/bin/netductor; then nd_sync /usr/local/bin; RESTORE=; else echo \"rollback binary failed\" >&2; rm -f \"$RESTORE\"; fi\n" +
		"  else\n" +
		"    systemctl disable netductor-secondary-agent 2>/dev/null || true\n" +
		"    rm -f /usr/local/bin/netductor; nd_sync /usr/local/bin\n" +
		"  fi\n" +
		"  rm -f /root/bundle.json\n" +
		"}\n" +
		"trap 'echo join:ERR line=$LINENO >&2; rollback; exit 1' ERR\n" +
		"systemctl stop netductor-secondary-agent 2>/dev/null || true\n" +
		"echo join:install_binary\n" +
		"install -m 755 \"$TD/$ASSET\" \"$NEW\"\n" +
		"nd_sync \"$NEW\"\n" +
		"mv -f \"$NEW\" /usr/local/bin/netductor\n" +
		"NEW=\n" +
		"nd_sync /usr/local/bin\n" +
		"echo join:write_bundle\n" +
		"echo '" + b64 + "' | base64 -d > /root/bundle.json\n" +
		"chmod 600 /root/bundle.json\n" +
		"test -s /root/bundle.json || { echo \"bundle empty\" >&2; exit 1; }\n" +
		"echo join:secondary_join\n" +
		"netductor secondary join /root/bundle.json\n" +
		"echo join:health\n" +
		"for i in 1 2 3 4 5 6 7 8; do\n" +
		"  SB=$(systemctl is-active sing-box 2>&1 || true)\n" +
		"  [ \"$SB\" = active ] && break\n" +
		"  sleep 2\n" +
		"done\n" +
		"AG=$(systemctl is-active netductor-secondary-agent 2>&1 || true)\n" +
		"[ \"$SB\" = active ] || { echo \"sing-box not active: $SB\" >&2; journalctl -u sing-box -n 25 --no-pager >&2 || true; exit 1; }\n" +
		"[ \"$AG\" = active ] || { echo \"agent not active: $AG\" >&2; journalctl -u netductor-secondary-agent -n 25 --no-pager >&2 || true; exit 1; }\n" +
		"ss -tlnp | grep -qE ':443([^0-9]|$)' || { echo \"nothing on :443\" >&2; ss -tlnp >&2 || true; exit 1; }\n" +
		"trap - ERR\n" +
		"rm -f /var/lib/netductor/netductor.bin.prev 2>/dev/null || true\n" +
		"nd_sync /var/lib/netductor\n" +
		"echo join:ok\n"
	out, err := runSSH(client, script)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("remote join failed: %s", trimLog(msg))
	}
	return "join ok\n", nil
}

func installMTLSMaterial(client *ssh.Client, ca, cert, key []byte) (string, error) {
	if len(ca) == 0 || len(cert) == 0 || len(key) == 0 {
		return "", fmt.Errorf("mTLS material incomplete (ca=%d cert=%d key=%d bytes)", len(ca), len(cert), len(key))
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, item := range []struct {
		name string
		data []byte
	}{
		{name: "ca.crt", data: ca},
		{name: "client.crt", data: cert},
		{name: "client.key", data: key},
	} {
		hdr := &tar.Header{Name: item.name, Mode: 0o600, Size: int64(len(item.data))}
		if err := tw.WriteHeader(hdr); err != nil {
			return "", err
		}
		if n, err := tw.Write(item.data); err != nil {
			return "", err
		} else if n != len(item.data) {
			return "", fmt.Errorf("short tar write for %s: %d/%d", item.name, n, len(item.data))
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	// Avoid SSH stdin races (tar on stdin can yield empty extract + confusing exit on some hosts).
	// Cert material is small; pass as base64 on the remote shell command.
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	script := `set -eu
P=/etc/netductor/secrets
TARGET="$P/mtls"
STAGE=
BACKUP=
HAD_OLD=0
nd_sync() { sync -f "$1" 2>/dev/null || sync 2>/dev/null || true; }
mkdir -p "$P"
test -d "$P" && test ! -L "$P" || { echo "mTLS secrets parent is not a real directory" >&2; exit 1; }
chmod 700 "$P"
STAGE=$(mktemp -d "$P/.mtls-stage.XXXXXX")
cleanup() {
  rc=$?
  if [ "$rc" -ne 0 ]; then
    if [ -n "${BACKUP:-}" ] && [ -d "$BACKUP" ]; then
      if { [ -e "$TARGET" ] || [ -L "$TARGET" ]; } && ! rm -rf "$TARGET"; then
        echo "CRITICAL: cannot clear partial mTLS target; backup remains at $BACKUP" >&2
        exit "$rc"
      fi
      if ! mv "$BACKUP" "$TARGET"; then
        echo "CRITICAL: mTLS rollback failed; backup remains at $BACKUP" >&2
        exit "$rc"
      fi
      nd_sync "$P"
    elif [ "${HAD_OLD:-0}" -eq 0 ]; then
      rm -rf "$TARGET" 2>/dev/null || echo "CRITICAL: cannot remove partial mTLS target" >&2
      nd_sync "$P"
    fi
  fi
  [ -z "${STAGE:-}" ] || rm -rf "$STAGE"
  exit "$rc"
}
trap cleanup EXIT
echo "` + b64 + `" | base64 -d | tar -x --no-same-owner -f - -C "$STAGE"
for f in ca.crt client.crt client.key; do
  test -f "$STAGE/$f" && test ! -L "$STAGE/$f" && test -s "$STAGE/$f"
  chmod 600 "$STAGE/$f"
  nd_sync "$STAGE/$f"
done
test "$(find "$STAGE" -mindepth 1 -maxdepth 1 -type f 2>/dev/null | wc -l | tr -d ' ')" -eq 3
chmod 700 "$STAGE"
nd_sync "$STAGE"
if [ -e "$TARGET" ] || [ -L "$TARGET" ]; then
  test -d "$TARGET" && test ! -L "$TARGET" || { echo "existing mTLS target is not a real directory" >&2; exit 1; }
  HAD_OLD=1
  BACKUP=$(mktemp -d "$P/.mtls-backup.XXXXXX")
  rmdir "$BACKUP"
  mv "$TARGET" "$BACKUP"
  nd_sync "$P"
fi
mv "$STAGE" "$TARGET"
STAGE=
nd_sync "$P"
trap - EXIT
if [ "$HAD_OLD" -eq 1 ]; then printf 'MTLS_BACKUP=%s\n' "$BACKUP"; else printf 'MTLS_BACKUP=-\n'; fi
`
	out, err := runSSH(client, script)
	if err != nil {
		return "", fmt.Errorf("mTLS atomic install failed: %w (%s)", err, trimLog(out))
	}
	const marker = "MTLS_BACKUP="
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, marker) {
			backup := strings.TrimSpace(strings.TrimPrefix(line, marker))
			if backup == "-" {
				return "-", nil
			}
			if !validMTLSBackupPath(backup) {
				return "", fmt.Errorf("remote returned invalid mTLS backup path %q", backup)
			}
			return backup, nil
		}
	}
	return "", fmt.Errorf("mTLS atomic install did not return transaction marker (out=%q)", trimLog(out))
}

func validMTLSBackupPath(path string) bool {
	const prefix = "/etc/netductor/secrets/.mtls-backup."
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	suf := path[len(prefix):]
	if len(suf) < 6 || len(suf) > 32 {
		return false
	}
	for _, r := range suf {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func finishMTLSMaterial(client *ssh.Client, backup string, restore bool) error {
	if backup == "" {
		return fmt.Errorf("mTLS transaction path missing")
	}
	if backup != "-" && !validMTLSBackupPath(backup) {
		return fmt.Errorf("invalid mTLS transaction path")
	}
	if backup == "-" && !restore {
		return nil
	}
	quoted := "'" + backup + "'"
	var script string
	if restore {
		if backup == "-" {
			script = `set -eu
P=/etc/netductor/secrets
TARGET="$P/mtls"
rm -rf "$TARGET"
sync -f "$P"
`
		} else {
			script = `set -eu
P=/etc/netductor/secrets
TARGET="$P/mtls"
BACKUP=` + quoted + `
test -d "$BACKUP" && test ! -L "$BACKUP"
rm -rf "$TARGET"
mv "$BACKUP" "$TARGET"
sync -f "$P"
`
		}
	} else {
		script = `set -eu
P=/etc/netductor/secrets
BACKUP=` + quoted + `
test -d "$BACKUP" && test ! -L "$BACKUP"
rm -rf "$BACKUP"
sync -f "$P"
`
	}
	out, err := runSSH(client, script)
	if err != nil {
		return fmt.Errorf("mTLS transaction finalize (restore=%v) failed: %w (%s)", restore, err, trimLog(out))
	}
	return nil
}

// ProvisionFromCore: password bootstrap → install pubkey → independent key-only gate → mTLS/join/harden on key session.
func ProvisionFromCore(in ProvisionIn, bundleJSON string) (*ProvisionResult, error) {
	var log strings.Builder
	keys, err := resolveInstallPubKeys(in)
	if err != nil {
		return nil, err
	}
	if len(in.MTLSCA) == 0 || len(in.MTLSCert) == 0 || len(in.MTLSKey) == 0 {
		return nil, fmt.Errorf("mTLS material required before secondary provision (ca=%d cert=%d key=%d)", len(in.MTLSCA), len(in.MTLSCert), len(in.MTLSKey))
	}

	// R6: reset TOFU only after all local prerequisites have passed. A malformed
	// key or incomplete mTLS input must not erase an existing trust pin.
	if in.ForceReinstall {
		if err := ForgetSSHHost(in.Host); err != nil {
			return nil, fmt.Errorf("ForceReinstall: cannot reset SSH trust for %s: %w", in.Host, err)
		}
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
	tr := NewSSHTransportBootstrap(in.Host, in.Port, in.User, in.Password, in.SSHPrivateKey, in.SSHKeyPassphrase)
	if err := tr.PromoteToKeyVerified(); err != nil {
		return nil, fmt.Errorf("%s: %w", StageTransportSwitch, err)
	}
	in.Password = ""
	log.WriteString(StageKeyOnlyVerification + ": ok; " + StageTransportSwitch + "=key_verified\n")

	mtlsBackup, err := installMTLSMaterial(client, in.MTLSCA, in.MTLSCert, in.MTLSKey)
	if err != nil {
		return nil, fmt.Errorf("mtls material: %w", err)
	}
	log.WriteString("mtls client material atomically installed; previous set retained\n")
	_, err = RemoteJoin(client, bundleJSON)
	log.WriteString("remote join completed\n")
	pubJoined := strings.Join(keys, "\n")
	if err != nil {
		restoreErr := finishMTLSMaterial(client, mtlsBackup, true)
		if restoreErr != nil {
			return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String()}, fmt.Errorf("secondary-install-components: join failed: %w; additionally %v (services remain stopped)", err, restoreErr)
		}
		log.WriteString("previous mTLS material restored after join failure\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String()}, fmt.Errorf("secondary-install-components: join failed: %w", err)
	}
	// Strict post-join checks (RemoteJoin already tests units; re-verify)
	_, sbErr := runSSH(client, `test "$(systemctl is-active sing-box)" = "active"`)
	_, agErr := runSSH(client, `test "$(systemctl is-active netductor-secondary-agent)" = "active"`)
	_, p443 := runSSH(client, `ss -tlnp | grep -qE ':443([^0-9]|$)'`)
	agentOK := agErr == nil
	singOK := sbErr == nil && p443 == nil
	if !agentOK || !singOK {
		_, _ = runSSH(client, "systemctl stop netductor-secondary-agent 2>/dev/null || true; systemctl stop sing-box 2>/dev/null || true")
		restoreErr := finishMTLSMaterial(client, mtlsBackup, true)
		if restoreErr != nil {
			return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: agentOK, SingBoxOK: singOK},
				fmt.Errorf("post-join services not healthy: agent_ok=%v singbox_ok=%v; mTLS restore failed: %v (services stopped)", agentOK, singOK, restoreErr)
		}
		log.WriteString("previous mTLS material restored after post-join health failure; services stopped\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: agentOK, SingBoxOK: singOK},
			fmt.Errorf("post-join services not healthy: agent_ok=%v singbox_ok=%v", agentOK, singOK)
	}
	if err := finishMTLSMaterial(client, mtlsBackup, false); err != nil {
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: agentOK, SingBoxOK: singOK},
			fmt.Errorf("join is healthy but cannot finalize mTLS backup cleanup: %w", err)
	}
	log.WriteString("mTLS transaction committed; previous material cleanup complete\n")
	// Harden prepare on current port (still 22 during initial): new port + rollback timer.
	if err := HardenSSHPrepare(client); err != nil {
		log.WriteString("harden prepare: " + err.Error() + "\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			err
	}
	log.WriteString("secondary-ssh-hardening-prepare: ok\n")
	_ = client.Close()

	// Verify key-only on hardened port before finalize.
	in.Port = hardening.DefaultSSHPort
	in.Password = ""
	day2, err := verifyKeyOnly(in)
	if err != nil {
		log.WriteString("secondary-new-port-verification: " + err.Error() + "\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			fmt.Errorf("secondary-new-port-verification: %w (rollback timer will restore SSH if still armed)", err)
	}
	log.WriteString("secondary-new-port-verification: ok\n")
	defer day2.Close()

	if err := HardenSSHFinalize(day2); err != nil {
		log.WriteString("harden finalize: " + err.Error() + "\n")
		// Rollback timer still armed — do not cancel.
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			err
	}
	log.WriteString(StageHardenFinalize + ": applied (rollback still armed)\n")
	_ = day2.Close()

	// Independent key-only AFTER finalize — only then cancel rollback.
	finalC, err := verifyKeyOnly(in)
	if err != nil {
		log.WriteString(StagePasswordAuthDisable + ": " + err.Error() + "\n")
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			fmt.Errorf("%s: post-finalize key-only failed (rollback timer remains armed): %w", StagePasswordAuthDisable, err)
	}
	if err := HardenSSHCancelRollback(finalC); err != nil {
		_ = finalC.Close()
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			err
	}
	if err := tr.PromoteToHardened(hardening.DefaultSSHPort); err != nil {
		_ = finalC.Close()
		return &ProvisionResult{Host: in.Host, PublicKey: pubJoined, Log: log.String(), AgentOK: true, SingBoxOK: true},
			fmt.Errorf("%s: transport state: %w", StageTransportSwitch, err)
	}
	// Non-SSH side effects only after access proven and rollback cancelled.
	// Host-agent cleanup is post-commit housekeeping. SSH hardening is already
	// finalized and its rollback timer cancelled; surfacing a purge error as a
	// provisioning failure would leave the operator journal at provision_started
	// and make a retry incorrectly attempt password SSH on port 22. Keep the
	// deployment resumable and report the cleanup failure without undoing access.
	if err := HardenSSHHostAgentPurge(finalC); err != nil {
		log.WriteString("host-agent-purge: warning (SSH hardening committed): " + trimLog(err.Error()) + "\n")
	}
	_ = finalC.Close()
	log.WriteString(StagePasswordAuthDisable + ": ok; transport=hardened; rollback cancelled\n")
	return &ProvisionResult{
		Host: in.Host, PublicKey: pubJoined, Log: log.String(),
		AgentOK: true, SingBoxOK: true,
	}, nil
}
