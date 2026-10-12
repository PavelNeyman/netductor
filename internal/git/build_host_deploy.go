package gitstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// BuildHostState is live inventory (heartbeat).
type BuildHostState struct {
	ID           string `json:"id"`
	Label        string `json:"label,omitempty"`
	SSHHost      string `json:"ssh_host"`
	SSHPort      int    `json:"ssh_port"`
	SSHUser      string `json:"ssh_user"`
	Token        string `json:"token,omitempty"` // agent auth (not logged)
	AgentVersion string `json:"agent_version,omitempty"`
	Online       bool   `json:"online"`
	LastSeen     string `json:"last_seen,omitempty"`
	Uname        string `json:"uname,omitempty"`
	Docker       bool   `json:"docker"`
	DiskFreeGB   float64 `json:"disk_free_gb,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	DeployedAt   string `json:"deployed_at,omitempty"`
}

func buildHostsStateDir() string {
	return filepath.Join(paths.StateDir(), "build-hosts")
}

func buildHostStatePath(id string) string {
	return filepath.Join(buildHostsStateDir(), sanitize(id)+".json")
}

// SaveBuildHostState persists inventory entry (token redacted in List for non-root callers handled by API).
func SaveBuildHostState(s BuildHostState) error {
	_ = os.MkdirAll(buildHostsStateDir(), 0o700)
	if s.ID == "" {
		s.ID = sanitize(s.SSHHost)
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(buildHostStatePath(s.ID), append(b, '\n'), 0o600)
}

// LoadBuildHostState by id or host.
func LoadBuildHostState(id string) (BuildHostState, error) {
	b, err := os.ReadFile(buildHostStatePath(sanitize(id)))
	if err != nil {
		return BuildHostState{}, err
	}
	var s BuildHostState
	err = json.Unmarshal(b, &s)
	return s, err
}

// ListBuildHostStates all registered hosts.
func ListBuildHostStates() []BuildHostState {
	ents, err := os.ReadDir(buildHostsStateDir())
	if err != nil {
		return nil
	}
	var out []BuildHostState
	now := time.Now().UTC()
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(buildHostsStateDir(), e.Name()))
		if err != nil {
			continue
		}
		var s BuildHostState
		if json.Unmarshal(b, &s) != nil {
			continue
		}
		s.Online = false
		if s.LastSeen != "" {
			if t, err := time.Parse(time.RFC3339, s.LastSeen); err == nil && now.Sub(t) < 90*time.Second {
				s.Online = true
			}
		}
		out = append(out, s)
	}
	return out
}

// PrimaryBuildHost returns first enabled config host state or empty.
func PrimaryBuildHostOnline() bool {
	for _, s := range ListBuildHostStates() {
		if s.Online {
			return true
		}
	}
	c := LoadBuildHost()
	if !c.Enabled {
		return false
	}
	s, err := LoadBuildHostState(sanitize(c.SSHHost))
	if err != nil {
		return false
	}
	if s.LastSeen == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, s.LastSeen)
	return err == nil && time.Since(t) < 90*time.Second
}

// RecordBuildHostHeartbeat updates state from agent.
func RecordBuildHostHeartbeat(token string, facts map[string]any) error {
	ents, err := os.ReadDir(buildHostsStateDir())
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(buildHostsStateDir(), e.Name()))
		if err != nil {
			continue
		}
		var s BuildHostState
		if json.Unmarshal(b, &s) != nil {
			continue
		}
		if s.Token == "" || s.Token != token {
			continue
		}
		s.LastSeen = time.Now().UTC().Format(time.RFC3339)
		s.Online = true
		if v, ok := facts["agent_version"].(string); ok {
			s.AgentVersion = v
		}
		if v, ok := facts["uname"].(string); ok {
			s.Uname = v
		}
		if v, ok := facts["docker"].(bool); ok {
			s.Docker = v
		}
		if v, ok := facts["disk_free_gb"].(float64); ok {
			s.DiskFreeGB = v
		}
		return SaveBuildHostState(s)
	}
	return fmt.Errorf("unknown build-host token")
}

func newBuildToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// DeployBuildHostOpts bootstrap from primary over SSH.
type DeployBuildHostOpts struct {
	Host     string
	User     string
	Port     int
	Password string // first install; empty if key already works
	KeyPath  string // local private key to use for SSH after/during
	Label    string
	PrimaryURL string // https://primary:8787 or public API base for agent
	AgentURL   string // download URL for netductor-agent-linux-amd64
}

// DeployBuildHost installs docker, agent ROLE=build, hardens SSH, registers state.
func DeployBuildHost(o DeployBuildHostOpts) (string, error) {
	var log strings.Builder
	if o.Host == "" {
		return "", fmt.Errorf("host required")
	}
	if o.User == "" {
		o.User = "root"
	}
	if o.Port == 0 {
		o.Port = 22 // first contact factory
	}
	if o.Label == "" {
		o.Label = "build-1"
	}
	if o.PrimaryURL == "" {
		o.PrimaryURL = "https://127.0.0.1:8787"
		// prefer public hostname if set
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_hostname")); err == nil {
			h := strings.TrimSpace(string(b))
			if h != "" {
				o.PrimaryURL = "https://" + h + ":8787"
			}
		}
	}
	token := newBuildToken()
	id := sanitize(o.Host)

	fmt.Fprintf(&log, "==> deploy build-host %s@%s:%d label=%s\n", o.User, o.Host, o.Port, o.Label)

	// 1) ensure key access: push pubkey if password
	pub := ""
	if o.KeyPath != "" {
		if b, err := os.ReadFile(o.KeyPath + ".pub"); err == nil {
			pub = strings.TrimSpace(string(b))
		}
	}
	if pub == "" {
		if b, err := os.ReadFile("/root/.ssh/id_ed25519.pub"); err == nil {
			pub = strings.TrimSpace(string(b))
		}
	}

	run := func(port int, cmd string) (string, error) {
		return sshRun(o.User, o.Host, port, o.KeyPath, o.Password, cmd)
	}

	// push key
	if pub != "" {
		push := fmt.Sprintf("mkdir -p /root/.ssh && chmod 700 /root/.ssh && grep -qxF %s /root/.ssh/authorized_keys 2>/dev/null || echo %s >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys",
			shellQuote(pub), shellQuote(pub))
		out, err := run(o.Port, push)
		log.WriteString(out)
		if err != nil {
			fmt.Fprintf(&log, "warn push key: %v\n", err)
		}
	}

	// packages + docker
	bootstrap := `set -e
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl git openssh-server
if ! command -v docker >/dev/null 2>&1; then
  apt-get install -y -qq docker.io || apt-get install -y -qq docker-ce || true
  systemctl enable --now docker 2>/dev/null || true
fi
mkdir -p /var/lib/netductor-build /etc/netductor-agent /usr/local/bin
`
	out, err := run(o.Port, bootstrap)
	log.WriteString(out)
	if err != nil {
		return log.String(), fmt.Errorf("bootstrap packages: %w", err)
	}

	// scp agent binary from local primary path
	agentSrc := "/usr/local/bin/netductor-agent"
	if _, err := os.Stat(agentSrc); err != nil {
		// try release path
		agentSrc = filepath.Join(paths.StateDir(), "releases", "netductor-agent-linux-amd64")
	}
	if _, err := os.Stat(agentSrc); err == nil {
		if err := scpFile(o.User, o.Host, o.Port, o.KeyPath, o.Password, agentSrc, "/usr/local/bin/netductor-agent"); err != nil {
			fmt.Fprintf(&log, "scp agent: %v\n", err)
		} else {
			_, _ = run(o.Port, "chmod 755 /usr/local/bin/netductor-agent")
		}
	} else {
		fmt.Fprintf(&log, "warn: no local agent binary; install manually or set AgentURL\n")
	}

	cfgBody := fmt.Sprintf("SERVER=%s\nTOKEN=%s\nDEVICE_ID=%s\nROLE=build\nINTERVAL=30\n",
		o.PrimaryURL, token, id)
	tmp := filepath.Join(os.TempDir(), "nd-bh-cfg-"+id)
	_ = os.WriteFile(tmp, []byte(cfgBody), 0o600)
	defer os.Remove(tmp)
	_ = scpFile(o.User, o.Host, o.Port, o.KeyPath, o.Password, tmp, "/etc/netductor-agent/config")

	unit := `[Unit]
Description=Netductor build agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/netductor-agent
Restart=always
RestartSec=5
MemoryMax=1G

[Install]
WantedBy=multi-user.target
`
	tmpU := filepath.Join(os.TempDir(), "nd-bh-unit-"+id)
	_ = os.WriteFile(tmpU, []byte(unit), 0o644)
	defer os.Remove(tmpU)
	_ = scpFile(o.User, o.Host, o.Port, o.KeyPath, o.Password, tmpU, "/etc/systemd/system/netductor-build-agent.service")
	out, _ = run(o.Port, "systemctl daemon-reload && systemctl enable --now netductor-build-agent.service && systemctl is-active netductor-build-agent.service")
	log.WriteString(out)

	// harden SSH to 52222 key-only if we have pub
	if pub != "" {
		hard := `set -e
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/99-netductor-build.conf << 'EOFC'
Port 52222
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
PermitRootLogin prohibit-password
EOFC
if ! grep -q '^Port 52222' /etc/ssh/sshd_config 2>/dev/null; then echo 'Port 52222' >> /etc/ssh/sshd_config; fi
sshd -t && systemctl reload sshd || systemctl reload ssh || true
`
		out, _ = run(o.Port, hard)
		log.WriteString(out)
	}

	// ufw allow from primary public ip if known
	if pip := strings.TrimSpace(readPrimaryPublicIP()); pip != "" {
		_, _ = run(o.Port, fmt.Sprintf("command -v ufw >/dev/null && ufw allow from %s to any port 52222 proto tcp || true", shellQuote(pip)))
	}

	sshPort := 52222
	if pub == "" {
		sshPort = o.Port
	}

	st := BuildHostState{
		ID: id, Label: o.Label, SSHHost: o.Host, SSHPort: sshPort, SSHUser: o.User,
		Token: token, DeployedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = SaveBuildHostState(st)

	cfg := LoadBuildHost()
	cfg.Enabled = true
	cfg.SSHHost = o.Host
	cfg.SSHPort = sshPort
	cfg.SSHUser = o.User
	cfg.IdentityFile = o.KeyPath
	cfg.Label = o.Label
	cfg.RemoteDir = "/var/lib/netductor-build"
	_ = SaveBuildHost(cfg)

	notify.AlertOnce("build-host:deployed:"+id, fmt.Sprintf("✅ Build-host <code>%s</code> deployed (%s)", o.Label, o.Host))
	fmt.Fprintf(&log, "ok id=%s token saved on primary; agent → %s\n", id, o.PrimaryURL)
	return log.String(), nil
}

func readPrimaryPublicIP() string {
	for _, p := range []string{
		filepath.Join(paths.EtcDir(), "secrets", "public_ip"),
		"/etc/netductor/secrets/public_ip",
	} {
		b, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

func sshRun(user, host string, port int, key, password, remoteCmd string) (string, error) {
	args := []string{
		"-p", strconv.Itoa(port),
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=25",
	}
	if key != "" && password == "" {
		args = append(args, "-i", key, "-o", "BatchMode=yes")
	} else if password != "" {
		args = append(args, "-o", "PreferredAuthentications=password", "-o", "PubkeyAuthentication=no")
	} else if key != "" {
		args = append(args, "-i", key, "-o", "BatchMode=yes")
	}
	args = append(args, user+"@"+host, remoteCmd)
	if password != "" {
		if _, err := exec.LookPath("sshpass"); err == nil {
			cmd := exec.Command("sshpass", append([]string{"-p", password, "ssh"}, args...)...)
			b, err := cmd.CombinedOutput()
			return string(b), err
		}
	}
	cmd := exec.Command("ssh", args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func scpFile(user, host string, port int, key, password, local, remote string) error {
	args := []string{"-P", strconv.Itoa(port), "-o", "StrictHostKeyChecking=accept-new"}
	if key != "" {
		args = append(args, "-i", key)
	}
	if password == "" {
		args = append(args, "-o", "BatchMode=yes")
	}
	args = append(args, local, fmt.Sprintf("%s@%s:%s", user, host, remote))
	if password != "" {
		if _, err := exec.LookPath("sshpass"); err == nil {
			cmd := exec.Command("sshpass", append([]string{"-p", password, "scp"}, args...)...)
			b, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%w: %s", err, string(b))
			}
			return nil
		}
	}
	cmd := exec.Command("scp", args...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(b))
	}
	return nil
}

// WatchBuildHosts probes via SSH and alerts when offline / low disk.
func WatchBuildHosts() {
	c := LoadBuildHost()
	if !c.Enabled || c.SSHHost == "" {
		return
	}
	list := ListBuildHostStates()
	if len(list) == 0 {
		list = []BuildHostState{{ID: sanitize(c.SSHHost), Label: c.Label, SSHHost: c.SSHHost, SSHPort: c.SSHPort, SSHUser: c.SSHUser}}
	}
	for _, s := range list {
		port := s.SSHPort
		if port == 0 {
			port = c.SSHPort
		}
		if port == 0 {
			port = 52222
		}
		user := s.SSHUser
		if user == "" {
			user = c.SSHUser
		}
		out, err := sshRun(user, s.SSHHost, port, c.IdentityFile, "",
			`systemctl is-active netductor-build-agent.service 2>/dev/null; df -BG /var/lib/netductor-build 2>/dev/null | tail -1; docker info >/dev/null 2>&1 && echo docker=yes || echo docker=no`)
		st := s
		if err != nil {
			st.Online = false
			st.LastError = err.Error()
			notify.AlertOnce("build-host:offline:"+st.ID,
				fmt.Sprintf("⚠️ Build-host <b>%s</b> unreachable", st.Label))
		} else {
			st.Online = true
			st.LastSeen = time.Now().UTC().Format(time.RFC3339)
			st.LastError = ""
			st.Docker = strings.Contains(out, "docker=yes")
			if !strings.Contains(out, "active") {
				notify.AlertOnce("build-host:agent:"+st.ID,
					fmt.Sprintf("⚠️ Build-host <b>%s</b> agent unit not active", st.Label))
			}
			for _, line := range strings.Split(out, "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 4 && strings.HasSuffix(fields[3], "G") {
					fmt.Sscanf(strings.TrimSuffix(fields[3], "G"), "%f", &st.DiskFreeGB)
				}
			}
			if st.DiskFreeGB > 0 && st.DiskFreeGB < 5 {
				notify.AlertOnce("build-host:disk:"+st.ID,
					fmt.Sprintf("⚠️ Build-host <b>%s</b> disk free %.1f GB", st.Label, st.DiskFreeGB))
			}
		}
		if prev, e := LoadBuildHostState(st.ID); e == nil {
			st.Token = prev.Token
			if st.Label == "" {
				st.Label = prev.Label
			}
			if st.DeployedAt == "" {
				st.DeployedAt = prev.DeployedAt
			}
		}
		_ = SaveBuildHostState(st)
	}
}
