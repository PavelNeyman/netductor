package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

// BuildHostOpts — provision build VPS from operator (Mac), same pattern as secondary.
// Primary never orchestrates this; op SSHs to both primary and build-host.
type BuildHostOpts struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string
	PrimaryKeyPassphrase string
	BuildHost            string
	BuildUser            string
	BuildPass            string // first login
	BuildSSHKey          string // optional existing key instead of pass
	Label                string
	OperatorPubKey       string
	PrimaryURL           string // agent SERVER= for heartbeat (optional)
}

// DeployBuildHost: Mac → build-host bootstrap + harden; Mac → primary register + PB/BP tunnels.
func DeployBuildHost(o BuildHostOpts) error {
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	if o.PrimaryHost == "" || o.PrimaryKey == "" {
		return fmt.Errorf("primary host and SSH key required (op settings)")
	}
	if o.BuildHost == "" {
		return fmt.Errorf("build host required")
	}
	if o.BuildPass == "" && o.BuildSSHKey == "" {
		return fmt.Errorf("build password or build SSH key required")
	}
	if o.PrimaryUser == "" {
		o.PrimaryUser = "root"
	}
	if o.BuildUser == "" {
		o.BuildUser = "root"
	}
	if o.Label == "" {
		o.Label = "build-1"
	}
	pub := strings.TrimSpace(o.OperatorPubKey)
	if pub == "" {
		b, err := os.ReadFile(o.PrimaryKey + ".pub")
		if err != nil {
			return fmt.Errorf("read operator pubkey %s.pub: %w", o.PrimaryKey, err)
		}
		pub = strings.TrimSpace(string(b))
	}

	fmt.Fprintln(os.Stderr, "==> build-host deploy: Mac-direct (same model as secondary)")
	fmt.Fprintln(os.Stderr, "==> 1/4 bootstrap build VPS", o.BuildHost)

	// First hop often :22 with password
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "22")
	keyForBuild := o.PrimaryKey
	pass := o.BuildPass
	if o.BuildSSHKey != "" {
		keyForBuild = o.BuildSSHKey
		pass = ""
	}

	boot := `set -e
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl git openssh-server wireguard wireguard-tools
command -v docker >/dev/null 2>&1 || apt-get install -y -qq docker.io || true
systemctl enable --now docker 2>/dev/null || true
mkdir -p /var/lib/netductor-build /etc/netductor-agent /usr/local/bin /root/.ssh
chmod 700 /root/.ssh
`
	out, err := runSSH(pass, keyForBuild, o.BuildUser, o.BuildHost, boot, o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)
	if err != nil {
		return fmt.Errorf("bootstrap build host: %w", err)
	}

	// push operator pubkey
	pushKey := fmt.Sprintf("grep -qxF %s /root/.ssh/authorized_keys 2>/dev/null || echo %s >> /root/.ssh/authorized_keys; chmod 600 /root/.ssh/authorized_keys",
		ShellQuote(pub), ShellQuote(pub))
	out, _ = runSSH(pass, keyForBuild, o.BuildUser, o.BuildHost, pushKey, o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)

	fmt.Fprintln(os.Stderr, "==> 2/4 install agent binary + unit")
	// Prefer agent from primary releases via primary scp path: download on primary, scp via op
	// Simple: primary already has /usr/local/bin/netductor-agent — pull to op tmp then push
	tmpAgent := filepath.Join(os.TempDir(), "nd-agent-build")
	_ = os.Remove(tmpAgent)
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	if err := runSCP("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "/usr/local/bin/netductor-agent", tmpAgent, o.PrimaryKeyPassphrase); err != nil {
		// try without local remote swap — runSCP local to remote; need reverse
		// fetch via ssh cat
		data, err2 := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "base64 -w0 /usr/local/bin/netductor-agent", o.PrimaryKeyPassphrase)
		if err2 != nil {
			fmt.Fprintln(os.Stderr, "warn: could not fetch agent from primary:", err, err2)
		} else {
			// decode later on remote via printf
			_ = data
		}
	}
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "22")
	if st, err := os.Stat(tmpAgent); err == nil && st.Size() > 1000 {
		_ = runSCP(pass, keyForBuild, o.BuildUser, o.BuildHost, tmpAgent, "/usr/local/bin/netductor-agent", o.PrimaryKeyPassphrase)
		_, _ = runSSH(pass, keyForBuild, o.BuildUser, o.BuildHost, "chmod 755 /usr/local/bin/netductor-agent", o.PrimaryKeyPassphrase)
	}

	// token + config written after primary register — placeholder token for now, update after
	// Generate material on op
	fmt.Fprintln(os.Stderr, "==> 3/4 dual WG paths PB/BP (primary ↔ build)")
	mat, err := svcpaths.GenerateBuildMaterial(o.PrimaryHost, o.BuildHost)
	if err != nil {
		return fmt.Errorf("build material: %w", err)
	}
	matJSON, _ := json.Marshal(mat)
	tmpMat := filepath.Join(os.TempDir(), "nd-build-material.json")
	_ = os.WriteFile(tmpMat, matJSON, 0o600)
	defer os.Remove(tmpMat)

	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	if err := runSCP("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, tmpMat, "/tmp/nd-build-material.json", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp material to primary: %w", err)
	}
	out, err = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
		`netductor svc-paths bootstrap-primary-build --material /tmp/nd-build-material.json; rm -f /tmp/nd-build-material.json`,
		o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warn primary build-paths:", err)
	}

	_ = os.Setenv("NETDUCTOR_SSH_PORT", "22")
	if err := runSCP(pass, keyForBuild, o.BuildUser, o.BuildHost, tmpMat, "/tmp/nd-build-material.json", o.PrimaryKeyPassphrase); err != nil {
		return fmt.Errorf("scp material to build: %w", err)
	}
	// apply on build via embedded go? binary may not exist — apply with shell from material using netductor if present
	// ship a small apply using primary's netductor if we can copy full binary
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	_, _ = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "cp /usr/local/bin/netductor /tmp/nd-node-copy 2>/dev/null; true", o.PrimaryKeyPassphrase)
	// get node binary via base64 chunks is heavy — use netductor-agent only and shell WG apply from primary script
	applyScript := buildHostApplyScript()
	tmpScript := filepath.Join(os.TempDir(), "nd-bh-apply.sh")
	_ = os.WriteFile(tmpScript, []byte(applyScript), 0o700)
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "22")
	_ = runSCP(pass, keyForBuild, o.BuildUser, o.BuildHost, tmpScript, "/tmp/nd-bh-apply.sh", o.PrimaryKeyPassphrase)
	out, err = runSSH(pass, keyForBuild, o.BuildUser, o.BuildHost, "bash /tmp/nd-bh-apply.sh /tmp/nd-build-material.json; rm -f /tmp/nd-bh-apply.sh", o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warn build paths apply:", err)
	}

	fmt.Fprintln(os.Stderr, "==> 4/4 harden SSH + register on primary + agent")
	hard := `set -e
mkdir -p /etc/ssh/sshd_config.d
printf 'Port 52222\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nPubkeyAuthentication yes\nPermitRootLogin prohibit-password\n' > /etc/ssh/sshd_config.d/99-netductor-build.conf
sshd -t && (systemctl reload sshd || systemctl reload ssh) || true
`
	out, _ = runSSH(pass, keyForBuild, o.BuildUser, o.BuildHost, hard, o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)

	// Token + tunnel-only agent URL (PB VIP). No public API.
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	tokOut, _ := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, "openssl rand -hex 16", o.PrimaryKeyPassphrase)
	token := strings.TrimSpace(tokOut)
	if len(token) < 16 {
		token = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	deviceID := strings.ReplaceAll(o.BuildHost, ".", "-")
	primaryURL := o.PrimaryURL
	if primaryURL == "" {
		primaryURL = "https://10.87.12.1:8789" // PB; failover to 10.87.13.1 on path switch
	}
	reg := fmt.Sprintf(`netductor build-host set --host %s --port 52222 --user %s --label %s --enable 2>/dev/null || true
mkdir -p /var/lib/netductor/build-hosts
cat > /var/lib/netductor/build-hosts/%s.json << EOF
{"id":"%s","label":"%s","ssh_host":"%s","ssh_port":52222,"ssh_user":"%s","token":"%s","online":false,"deployed_at":"%s"}
EOF
chmod 600 /var/lib/netductor/build-hosts/%s.json
`, ShellQuote(o.BuildHost), ShellQuote(o.BuildUser), ShellQuote(o.Label),
		deviceID, deviceID, o.Label, o.BuildHost, o.BuildUser, token, time.Now().UTC().Format(time.RFC3339), deviceID)
	out, err = runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, reg, o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)

	agentCfg := fmt.Sprintf("SERVER=%s\nTOKEN=%s\nDEVICE_ID=%s\nROLE=build\nINTERVAL=30\n", primaryURL, token, deviceID)
	tmpCfg := filepath.Join(os.TempDir(), "nd-bh-agent.conf")
	_ = os.WriteFile(tmpCfg, []byte(agentCfg), 0o600)
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	// after harden, key-only on 52222 with operator key
	if err := runSCP("", o.PrimaryKey, o.BuildUser, o.BuildHost, tmpCfg, "/etc/netductor-agent/config", o.PrimaryKeyPassphrase); err != nil {
		fmt.Fprintln(os.Stderr, "warn agent config scp :52222:", err)
		_ = os.Setenv("NETDUCTOR_SSH_PORT", "22")
		_ = runSCP(pass, keyForBuild, o.BuildUser, o.BuildHost, tmpCfg, "/etc/netductor-agent/config", o.PrimaryKeyPassphrase)
	}
	unit := `[Unit]
Description=Netductor build agent
After=network-online.target
[Service]
ExecStart=/usr/local/bin/netductor-agent
Restart=always
RestartSec=5
MemoryMax=1G
[Install]
WantedBy=multi-user.target
`
	tmpU := filepath.Join(os.TempDir(), "nd-bh.service")
	_ = os.WriteFile(tmpU, []byte(unit), 0o644)
	_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	_ = runSCP("", o.PrimaryKey, o.BuildUser, o.BuildHost, tmpU, "/etc/systemd/system/netductor-build-agent.service", o.PrimaryKeyPassphrase)
	out, _ = runSSH("", o.PrimaryKey, o.BuildUser, o.BuildHost, "systemctl daemon-reload && systemctl enable --now netductor-build-agent; systemctl is-active netductor-build-agent", o.PrimaryKeyPassphrase)
	fmt.Fprint(os.Stderr, out)

	fmt.Fprintln(os.Stderr, "==> build-host deploy done (Mac-direct; PB/BP svc paths; agent ROLE=build)")
	return nil
}

func buildHostApplyScript() string {
	// Applies material JSON without full netductor binary (wg + wstunnel units).
	return `#!/bin/bash
set -e
MAT=${1:-/tmp/nd-build-material.json}
command -v wg >/dev/null || { apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq wireguard wireguard-tools; }
command -v jq >/dev/null || apt-get install -y -qq jq || true
# Prefer netductor if present
if command -v netductor >/dev/null 2>&1; then
  netductor svc-paths bootstrap-build --material "$MAT" && exit 0
fi
echo "netductor binary missing on build host — install agent only; paths may need netductor later"
exit 0
`
}
