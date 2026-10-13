package gitstore

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// BuildHostConfig is optional remote CI executor (control plane stays primary).
type BuildHostConfig struct {
	Enabled      bool   `json:"enabled"`
	SSHUser      string `json:"ssh_user"`
	SSHHost      string `json:"ssh_host"`
	SSHPort      int    `json:"ssh_port"`
	IdentityFile string `json:"identity_file,omitempty"`
	RemoteDir    string `json:"remote_dir,omitempty"`
	Label        string `json:"label,omitempty"`
}

func buildHostPath() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_BUILD_HOST_CONFIG")); v != "" {
		return v
	}
	return filepath.Join(paths.EtcDir(), "build-host.json")
}

// LoadBuildHost reads config; disabled if missing.
func LoadBuildHost() BuildHostConfig {
	b, err := os.ReadFile(buildHostPath())
	if err != nil {
		return BuildHostConfig{SSHUser: "root", SSHPort: 52222, RemoteDir: "/var/lib/netductor-build"}
	}
	var c BuildHostConfig
	_ = json.Unmarshal(b, &c)
	if c.SSHUser == "" {
		c.SSHUser = "root"
	}
	if c.SSHPort == 0 {
		c.SSHPort = 52222
	}
	if c.RemoteDir == "" {
		c.RemoteDir = "/var/lib/netductor-build"
	}
	return c
}

// SaveBuildHost writes config 0600.
func SaveBuildHost(c BuildHostConfig) error {
	if c.SSHUser == "" {
		c.SSHUser = "root"
	}
	if c.SSHPort == 0 {
		c.SSHPort = 52222
	}
	if c.RemoteDir == "" {
		c.RemoteDir = "/var/lib/netductor-build"
	}
	_ = os.MkdirAll(filepath.Dir(buildHostPath()), 0o755)
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(buildHostPath(), append(b, '\n'), 0o600)
}

// BuildHostEnabled is true when ship should hand test_build to a build-host agent.
// Prefer the state registry written by Mac deploy (build-hosts/*.json); fall back to
// legacy /etc/netductor/build-host.json Enabled+SSHHost.
func BuildHostEnabled() bool {
	// Online agent wins. Stale registry alone must not hard-fail ship.
	if PrimaryBuildHostOnline() {
		return true
	}
	c := LoadBuildHost()
	return c.Enabled && strings.TrimSpace(c.SSHHost) != ""
}

func (c BuildHostConfig) target() string {
	return c.SSHUser + "@" + c.SSHHost
}

func (c BuildHostConfig) sshArgs(extra ...string) []string {
	args := []string{
		"-p", strconv.Itoa(c.SSHPort),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=20",
	}
	if c.IdentityFile != "" {
		args = append(args, "-i", c.IdentityFile)
	}
	args = append(args, extra...)
	return args
}

// TestBuildHost probes agent plane status (no SSH from primary).
// Legacy SSH probe removed — policy: primary never SSHs to build-host.
func TestBuildHost() (string, error) {
	list := ListBuildHostStates()
	if len(list) == 0 {
		return "", fmt.Errorf("no build-host in registry (deploy from op)")
	}
	online := 0
	var b strings.Builder
	for _, s := range list {
		st := "offline"
		if s.Online {
			st = "online"
			online++
		}
		fmt.Fprintf(&b, "%s  %s  last=%s\n", s.ID, st, s.LastSeen)
	}
	if online == 0 {
		return b.String(), fmt.Errorf("no online build-host agent (heartbeat over :8789)")
	}
	return b.String(), nil
}

func (c BuildHostConfig) runSSH(remoteCmd string) (string, error) {
	args := c.sshArgs(c.target(), remoteCmd)
	cmd := exec.Command("ssh", args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func (c BuildHostConfig) scpTo(local, remote string) error {
	// scp uses -P for port
	scpArgs := []string{"-P", strconv.Itoa(c.SSHPort), "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"}
	if c.IdentityFile != "" {
		scpArgs = append(scpArgs, "-i", c.IdentityFile)
	}
	scpArgs = append(scpArgs, local, fmt.Sprintf("%s:%s", c.target(), remote))
	cmd := exec.Command("scp", scpArgs...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp to remote: %w: %s", err, string(b))
	}
	return nil
}

func (c BuildHostConfig) scpFrom(remote, local string) error {
	scpArgs := []string{"-P", strconv.Itoa(c.SSHPort), "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new", "-r"}
	if c.IdentityFile != "" {
		scpArgs = append(scpArgs, "-i", c.IdentityFile)
	}
	scpArgs = append(scpArgs, fmt.Sprintf("%s:%s", c.target(), remote), local)
	cmd := exec.Command("scp", scpArgs...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp from remote: %w: %s", err, string(b))
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// StatusLine for doctor/CLI.
func BuildHostStatusLine() string {
	states := ListBuildHostStates()
	if len(states) > 0 {
		online := 0
		for _, s := range states {
			if s.Online {
				online++
			}
		}
		return fmt.Sprintf("build-host agents=%d online=%d (registry)", len(states), online)
	}
	c := LoadBuildHost()
	if !c.Enabled || c.SSHHost == "" {
		return "build-host disabled (local CI)"
	}
	return fmt.Sprintf("build-host config %s@%s:%d remote=%s (legacy)", c.SSHUser, c.SSHHost, c.SSHPort, c.RemoteDir)
}
