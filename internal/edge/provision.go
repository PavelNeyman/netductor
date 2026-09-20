package edge

import (
	"encoding/base64"
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
	Password       string
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

	sshBase := []string{"-o", "StrictHostKeyChecking=accept-new"}
	if strings.TrimSpace(opts.Password) == "" {
		sshBase = append(sshBase, "-o", "BatchMode=yes")
	} else {
		sshBase = append(sshBase, "-o", "PreferredAuthentications=password", "-o", "PubkeyAuthentication=no")
	}
	if opts.SSHKey != "" {
		sshBase = append(sshBase, "-i", opts.SSHKey)
	}
	scpArgs := append(append([]string{"scp"}, sshBase...), agent, opts.SSHTarget+":/tmp/netductor-agent")
	if out, err := sshpassCmd(opts.Password, scpArgs).CombinedOutput(); err != nil {
		return fmt.Errorf("scp: %s %v", out, err)
	}
	cfg := fmt.Sprintf("SERVER=%s\nTOKEN=%s\nDEVICE_ID=%s\nINTERVAL=60\n",
		server, boot, opts.DeviceID)

	mtlsBlock := ""
	if len(opts.MTLSCA) > 0 && len(opts.MTLSCert) > 0 && len(opts.MTLSKey) > 0 {
		enc := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
		mtlsBlock = fmt.Sprintf(`
mkdir -p /etc/netductor-agent/mtls
echo '%s' | base64 -d > /etc/netductor-agent/mtls/ca.crt
echo '%s' | base64 -d > /etc/netductor-agent/mtls/client.crt
echo '%s' | base64 -d > /etc/netductor-agent/mtls/client.key
chmod 700 /etc/netductor-agent/mtls
chmod 600 /etc/netductor-agent/mtls/*
`, enc(opts.MTLSCA), enc(opts.MTLSCert), enc(opts.MTLSKey))
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
`, cfg, mtlsBlock, harden)
	sshArgs := append(append([]string{"ssh"}, sshBase...), opts.SSHTarget, "sh", "-s")
	cmd := sshpassCmd(opts.Password, sshArgs)
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh: %s %v", out, err)
	}
	return nil
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
	if password == "" {
		return exec.Command(args[0], args[1:]...)
	}
	sp, err := exec.LookPath("sshpass")
	if err != nil {
		return exec.Command(args[0], args[1:]...)
	}
	return exec.Command(sp, append([]string{"-p", password}, args...)...)
}
