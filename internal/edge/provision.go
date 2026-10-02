package edge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type ProvisionOpts struct {
	SSHTarget      string // root@192.168.1.1
	DeviceID       string
	ServerURL      string // prefer https://primary:8789
	AgentBin       string
	SSHKey         string
	Arch           string
	Token          string
	Password       string // current root password; empty = factory OpenWrt (empty password login)
	NewRootPassword string // optional: set root password before disabling password SSH
	OperatorPubKey string
	// Optional mTLS client material (from primary EnsureClientFor).
	MTLSCA   []byte
	MTLSCert []byte
	MTLSKey  []byte
}

func Provision(opts ProvisionOpts) error {
	if opts.SSHTarget == "" || opts.DeviceID == "" || opts.ServerURL == "" {
		return fmt.Errorf("ssh target, device_id and server url required")
	}
	boot := strings.TrimSpace(opts.Token)
	if boot == "" {
		boot = BootstrapToken()
	}
	if boot == "" {
		return fmt.Errorf("edge_bootstrap_token missing (pass Token or run on primary)")
	}
	agent := opts.AgentBin
	if agent == "" {
		agent = filepath.Join(paths.OptDir(), "bin", "netductor-agent")
		if _, err := os.Stat(agent); err != nil {
			agent = "/usr/local/bin/netductor-agent"
		}
	}
	if _, err := os.Stat(agent); err != nil {
		return fmt.Errorf("agent binary not found: %s (download release asset first)", agent)
	}

	// Prefer mTLS agent plane URL.
	server := strings.TrimRight(opts.ServerURL, "/")
	if len(opts.MTLSCert) > 0 {
		server = forceAgentMTLSURL(server)
	}

	// Factory OpenWrt often has empty root password. Prefer password auth (including empty)
	// for first contact; do not BatchMode+operator key (pubkey not on router yet).
	sshBase := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "PreferredAuthentications=password",
		"-o", "PubkeyAuthentication=no",
		"-o", "NumberOfPasswordPrompts=3",
	}
	// Re-provision path: no password attempt, operator key already on router.
	useKeyOnly := strings.TrimSpace(opts.Password) == "" && opts.SSHKey != "" && opts.NewRootPassword == "" && os.Getenv("NETDUCTOR_EDGE_KEY_ONLY") == "1"
	if useKeyOnly {
		sshBase = []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes", "-i", opts.SSHKey}
	}
	// OpenWrt/Dropbear: modern OpenSSH scp uses SFTP subsystem → "subsystem request failed".
	// Prefer ssh + stdin (always works). Fallback: scp -O (legacy SCP protocol).
	if err := putFileSSH(opts.Password, sshBase, opts.SSHTarget, agent, "/tmp/netductor-agent"); err != nil {
		if opts.SSHKey != "" && !useKeyOnly {
			keyBase := []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes", "-i", opts.SSHKey}
			if err2 := putFileSSH("", keyBase, opts.SSHTarget, agent, "/tmp/netductor-agent"); err2 != nil {
				return fmt.Errorf("put agent: %v (key fallback: %v); empty password = factory OpenWrt, leave router password blank", err, err2)
			}
			sshBase = keyBase
		} else {
			return fmt.Errorf("put agent: %w", err)
		}
	}
	cfg := fmt.Sprintf("SERVER=%s\nTOKEN=%s\nDEVICE_ID=%s\nINTERVAL=60\n",
		server, boot, opts.DeviceID)

	// OpenWrt often has no base64 applet — push binary material via ssh stdin, not echo|base64 -d.
	mtlsBlock := ""
	if len(opts.MTLSCA) > 0 && len(opts.MTLSCert) > 0 && len(opts.MTLSKey) > 0 {
		_ = runSSH(opts.Password, sshBase, opts.SSHTarget, "mkdir -p /etc/netductor-agent/mtls && chmod 700 /etc/netductor-agent/mtls")
		if err := putBytesSSH(opts.Password, sshBase, opts.SSHTarget, opts.MTLSCA, "/etc/netductor-agent/mtls/ca.crt"); err != nil {
			return fmt.Errorf("put mtls ca: %w", err)
		}
		if err := putBytesSSH(opts.Password, sshBase, opts.SSHTarget, opts.MTLSCert, "/etc/netductor-agent/mtls/client.crt"); err != nil {
			return fmt.Errorf("put mtls cert: %w", err)
		}
		if err := putBytesSSH(opts.Password, sshBase, opts.SSHTarget, opts.MTLSKey, "/etc/netductor-agent/mtls/client.key"); err != nil {
			return fmt.Errorf("put mtls key: %w", err)
		}
		mtlsBlock = "chmod 600 /etc/netductor-agent/mtls/* 2>/dev/null || true\n"
	}

	// Optional root password: file push avoids base64 and shell metachar issues.
	setPass := ""
	if np := strings.TrimSpace(opts.NewRootPassword); np != "" {
		if err := putBytesSSH(opts.Password, sshBase, opts.SSHTarget, []byte(np), "/tmp/nd-root-pass"); err != nil {
			return fmt.Errorf("put root pass: %w", err)
		}
		setPass = `
# set root password (optional, before harden) — no base64 needed
if [ -f /tmp/nd-root-pass ]; then
  NEWP=$(cat /tmp/nd-root-pass)
  rm -f /tmp/nd-root-pass
  if [ -n "$NEWP" ]; then
    printf '%s\n%s\n' "$NEWP" "$NEWP" | passwd root 2>/dev/null || \
      (command -v chpasswd >/dev/null && printf 'root:%s\n' "$NEWP" | chpasswd) || true
  fi
fi
`
	}

	harden := ""
	if pub := strings.TrimSpace(opts.OperatorPubKey); pub != "" {
		esc := strings.ReplaceAll(pub, "'", `'"'"'`)
		harden = fmt.Sprintf(`
mkdir -p /root/.ssh
chmod 700 /root/.ssh
touch /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
grep -qxF '%s' /root/.ssh/authorized_keys || echo '%s' >> /root/.ssh/authorized_keys
mkdir -p /etc/dropbear
touch /etc/dropbear/authorized_keys
chmod 600 /etc/dropbear/authorized_keys
grep -qxF '%s' /etc/dropbear/authorized_keys || echo '%s' >> /etc/dropbear/authorized_keys
if [ -f /etc/config/dropbear ] && command -v uci >/dev/null 2>&1; then
  uci set dropbear.@dropbear[0].PasswordAuth='off' 2>/dev/null || true
  uci set dropbear.@dropbear[0].RootPasswordAuth='off' 2>/dev/null || true
  uci commit dropbear 2>/dev/null || true
  /etc/init.d/dropbear reload 2>/dev/null || /etc/init.d/dropbear restart 2>/dev/null || true
fi
if [ -d /etc/ssh ]; then
  mkdir -p /etc/ssh/sshd_config.d
  cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'SSH_EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin prohibit-password
PubkeyAuthentication yes
SSH_EOF
  /etc/init.d/sshd reload 2>/dev/null || systemctl reload ssh 2>/dev/null || true
fi
`, esc, esc, esc, esc)
	}

	script := fmt.Sprintf(`set -e
mkdir -p /etc/netductor-agent /usr/sbin
mv /tmp/netductor-agent /usr/sbin/netductor-agent
chmod 755 /usr/sbin/netductor-agent
cat > /etc/netductor-agent/config <<'CFG'
%s
CFG
chmod 600 /etc/netductor-agent/config
%s
%s
%s
if [ -d /etc/init.d ]; then
  cat > /etc/init.d/netductor-agent <<'INIT'
#!/bin/sh /etc/rc.common
START=99
USE_PROCD=1
start_service() {
  procd_open_instance
  procd_set_param command /usr/sbin/netductor-agent
  procd_set_param respawn
  procd_set_param file /etc/netductor-agent/config
  procd_close_instance
}
INIT
  chmod 755 /etc/init.d/netductor-agent
  /etc/init.d/netductor-agent enable 2>/dev/null || true
  /etc/init.d/netductor-agent restart 2>/dev/null || /usr/sbin/netductor-agent &
fi
`, cfg, mtlsBlock, setPass, harden)
	sshArgs := append(append([]string{"ssh"}, sshBase...), opts.SSHTarget, "sh", "-s")
	cmd := sshpassCmd(opts.Password, sshArgs)
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh: %s %v", cleanSSHNoise(string(out)), err)
	}
	return nil
}

func cleanSSHNoise(s string) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.Contains(l, "post-quantum key exchange") || strings.Contains(l, "store now, decrypt later") || strings.Contains(l, "openssh.com/pq.html") {
			continue
		}
		if strings.HasPrefix(l, "**") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func forceAgentMTLSURL(server string) string {
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	rest := strings.TrimPrefix(strings.TrimPrefix(server, "https://"), "http://")
	host := rest
	if i := strings.Index(rest, "/"); i >= 0 {
		host = rest[:i]
	}
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return "https://" + host + ":8789"
}

func sshpassCmd(password string, args []string) *exec.Cmd {
	// Empty password is valid for factory OpenWrt; still use sshpass when available.
	sp, err := exec.LookPath("sshpass")
	if err != nil {
		return exec.Command(args[0], args[1:]...)
	}
	// Prefer env form so empty password works (sshpass -e).
	cmd := exec.Command(sp, append([]string{"-e"}, args...)...)
	cmd.Env = append(os.Environ(), "SSHPASS="+password)
	return cmd
}


// putFileSSH copies localPath to remotePath via ssh stdin (Dropbear-safe; no SFTP subsystem).
func putFileSSH(password string, sshBase []string, target, localPath, remotePath string) error {
	b, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	return putBytesSSH(password, sshBase, target, b, remotePath)
}

func putBytesSSH(password string, sshBase []string, target string, data []byte, remotePath string) error {
	remoteCmd := "cat > " + shellQuotePath(remotePath)
	args := append(append([]string{}, sshBase...), target, remoteCmd)
	cmd := sshpassCmd(password, append([]string{"ssh"}, args...))
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-pipe put %s: %s (%v)", remotePath, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func runSSH(password string, sshBase []string, target, remoteCmd string) error {
	args := append(append([]string{}, sshBase...), target, remoteCmd)
	cmd := sshpassCmd(password, append([]string{"ssh"}, args...))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh: %s (%v)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func shellQuotePath(p string) string {
	return "'" + strings.ReplaceAll(p, "'", `'"'"'`) + "'"
}
