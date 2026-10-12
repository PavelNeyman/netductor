package gitstore

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// BuildHostEnabled is true when remote executor should be used for VPS ship test_build.
func BuildHostEnabled() bool {
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

// TestBuildHost SSH connectivity + remote mkdir.
func TestBuildHost() (string, error) {
	c := LoadBuildHost()
	if strings.TrimSpace(c.SSHHost) == "" {
		return "", fmt.Errorf("build-host host not set")
	}
	cmd := fmt.Sprintf("mkdir -p %s && uname -a && (command -v docker || command -v podman || echo 'no-container-engine')", shellQuote(c.RemoteDir))
	out, err := c.runSSH(cmd)
	if err != nil {
		return out, err
	}
	return out, nil
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

// RunRemoteTestBuild packs worktree on primary, builds on host, pulls artifacts.
func RunRemoteTestBuild(job *ShipJob, workDir string, logb *strings.Builder) error {
	c := LoadBuildHost()
	if !c.Enabled || c.SSHHost == "" {
		return fmt.Errorf("build-host not enabled")
	}
	fmt.Fprintf(logb, "build-host %s@%s:%d label=%s\n", c.SSHUser, c.SSHHost, c.SSHPort, c.Label)

	remoteJob := filepath.Join(c.RemoteDir, "jobs", sanitize(job.ID))
	remoteSrc := filepath.Join(remoteJob, "src")
	remoteArt := filepath.Join(remoteJob, "artifacts")
	_, _ = c.runSSH(fmt.Sprintf("rm -rf %s && mkdir -p %s %s", shellQuote(remoteJob), shellQuote(remoteSrc), shellQuote(remoteArt)))

	// pack source
	tarPath := filepath.Join(os.TempDir(), "nd-ship-"+sanitize(job.ID)+".tar.gz")
	defer os.Remove(tarPath)
	pack := exec.Command("tar", "-czf", tarPath, "-C", workDir, ".")
	if b, err := pack.CombinedOutput(); err != nil {
		return fmt.Errorf("pack worktree: %w: %s", err, string(b))
	}
	if err := c.scpTo(tarPath, filepath.Join(remoteJob, "src.tar.gz")); err != nil {
		return err
	}
	_, err := c.runSSH(fmt.Sprintf("tar -xzf %s -C %s", shellQuote(filepath.Join(remoteJob, "src.tar.gz")), shellQuote(remoteSrc)))
	if err != nil {
		return fmt.Errorf("remote extract: %w", err)
	}

	// secrets env file (remote, deleted after)
	var envLines []string
	envLines = append(envLines, LoadCISecretsEnv(job.Project)...)
	envFile := filepath.Join(os.TempDir(), "nd-env-"+sanitize(job.ID))
	_ = os.WriteFile(envFile, []byte(strings.Join(envLines, "\n")+"\n"), 0o600)
	defer os.Remove(envFile)
	_ = c.scpTo(envFile, filepath.Join(remoteJob, "env"))

	// remote build script: prefer docker golang image if Dockerfile absent
	script := fmt.Sprintf(`set -e
cd %s
if [ -f Dockerfile ]; then
  docker build -t nd-ship-%s . 2>&1 | tail -20
elif [ -f go.mod ]; then
  docker run --rm -v "$PWD":/src -w /src -e GOCACHE=/tmp/gocache -e GOMODCACHE=/tmp/gomod golang:1.22-bookworm bash -lc 'go test ./... && go build -o /src/artifacts/ ./...' 2>&1
  mkdir -p artifacts
  cp -a bin/. artifacts/ 2>/dev/null || true
  ls -la artifacts || true
else
  # generic: run make test/build if present
  if [ -f Makefile ]; then make test all 2>&1 || make 2>&1; fi
  mkdir -p artifacts
  find . -maxdepth 2 -type f -executable -path ./bin/\* -exec cp -a {} artifacts/ \; 2>/dev/null || true
fi
`, shellQuote(remoteSrc), sanitize(job.ID)[:12])

	scriptPath := filepath.Join(os.TempDir(), "nd-build-"+sanitize(job.ID)+".sh")
	_ = os.WriteFile(scriptPath, []byte(script), 0o700)
	defer os.Remove(scriptPath)
	_ = c.scpTo(scriptPath, filepath.Join(remoteJob, "run.sh"))

	out, err := c.runSSH(fmt.Sprintf("chmod +x %s && set -a && [ -f %s ] && . %s; set +a; bash %s",
		shellQuote(filepath.Join(remoteJob, "run.sh")),
		shellQuote(filepath.Join(remoteJob, "env")),
		shellQuote(filepath.Join(remoteJob, "env")),
		shellQuote(filepath.Join(remoteJob, "run.sh")),
	))
	logb.WriteString(out)
	if err != nil {
		return fmt.Errorf("remote build: %w", err)
	}

	// pull artifacts
	localArt := ShipJobArtifactsDir(job.ID)
	_ = os.MkdirAll(localArt, 0o755)
	_ = c.scpFrom(remoteArt+"/.", localArt+"/")
	// cleanup remote job dir (keep last fails optional)
	_, _ = c.runSSH(fmt.Sprintf("rm -rf %s", shellQuote(remoteJob)))
	fmt.Fprintf(logb, "build-host artifacts → %s at %s\n", localArt, time.Now().UTC().Format(time.RFC3339))
	return nil
}

// StatusLine for doctor/CLI.
func BuildHostStatusLine() string {
	c := LoadBuildHost()
	if !c.Enabled || c.SSHHost == "" {
		return "build-host disabled (local CI)"
	}
	return fmt.Sprintf("build-host enabled %s@%s:%d remote=%s", c.SSHUser, c.SSHHost, c.SSHPort, c.RemoteDir)
}
