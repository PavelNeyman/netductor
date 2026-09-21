// Package registry — thin local OCI registry (distribution/registry:2) + crane helper.
// Default bind 127.0.0.1:5000 (SSH tunnel / localhost only). Not a public harbor.
package registry

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

const (
	containerName = "netductor-registry"
	imageName     = "registry:2"
)

func DataDir() string {
	if v := os.Getenv("NETDUCTOR_REGISTRY_DATA"); v != "" {
		return v
	}
	return filepath.Join(paths.StateDir(), "registry")
}

// Addr host:port inside published mapping (default 127.0.0.1:5000).
func Addr() string {
	if v := os.Getenv("NETDUCTOR_REGISTRY_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:5000"
}

func HostPort() (host, port string) {
	a := Addr()
	h, p, err := net.SplitHostPort(a)
	if err != nil {
		return "127.0.0.1", "5000"
	}
	return h, p
}

func dockerBin() string {
	for _, c := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return ""
}

func CranePath() string {
	if p, err := exec.LookPath("crane"); err == nil {
		return p
	}
	cand := "/usr/local/bin/crane"
	if st, err := os.Stat(cand); err == nil && st.Mode().IsRegular() {
		return cand
	}
	return ""
}

type Status struct {
	OK          bool   `json:"ok"`
	Engine      string `json:"engine,omitempty"`
	Container   string `json:"container"`
	Running     bool   `json:"running"`
	Addr        string `json:"addr"`
	DataDir     string `json:"data_dir"`
	HTTPReach   bool   `json:"http_reachable"`
	CranePath   string `json:"crane,omitempty"`
	Hint        string `json:"hint,omitempty"`
	Error       string `json:"error,omitempty"`
}

func StatusInfo() Status {
	s := Status{
		Container: containerName,
		Addr:      Addr(),
		DataDir:   DataDir(),
		CranePath: CranePath(),
	}
	eng := dockerBin()
	s.Engine = eng
	if eng == "" {
		s.Error = "docker/podman not found"
		s.Hint = "install docker.io or podman, then: netductor registry ensure"
		return s
	}
	out, err := exec.Command(eng, "inspect", "-f", "{{.State.Running}}", containerName).CombinedOutput()
	running := err == nil && strings.TrimSpace(string(out)) == "true"
	s.Running = running
	if running {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://" + Addr() + "/v2/")
		if err == nil {
			_ = resp.Body.Close()
			s.HTTPReach = resp.StatusCode == 200 || resp.StatusCode == 401
		}
	}
	s.OK = s.Running && s.HTTPReach
	if s.OK {
		s.Hint = fmt.Sprintf("crane push/pull via %s (e.g. crane push ./img.tar %s/myapp:latest)", Addr(), Addr())
	} else if eng != "" && !running {
		s.Hint = "netductor registry ensure"
	}
	return s
}

// Ensure starts registry:2 published on Addr() with persistent data volume.
func Ensure() (Status, error) {
	eng := dockerBin()
	if eng == "" {
		return StatusInfo(), fmt.Errorf("docker or podman required")
	}
	_ = os.MkdirAll(DataDir(), 0o700)
	host, port := HostPort()

	// already running?
	st := StatusInfo()
	if st.Running && st.HTTPReach {
		return st, nil
	}
	if st.Running && !st.HTTPReach {
		_ = exec.Command(eng, "rm", "-f", containerName).Run()
	}

	// pull + run
	_ = exec.Command(eng, "pull", imageName).Run()
	args := []string{
		"run", "-d", "--name", containerName, "--restart", "unless-stopped",
		"-p", host + ":" + port + ":5000",
		"-v", DataDir() + ":/var/lib/registry",
		imageName,
	}
	// remove stopped container with same name
	_ = exec.Command(eng, "rm", "-f", containerName).Run()
	out, err := exec.Command(eng, args...).CombinedOutput()
	if err != nil {
		return StatusInfo(), fmt.Errorf("run registry: %w: %s", err, strings.TrimSpace(string(out)))
	}
	// wait ready
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		st = StatusInfo()
		if st.HTTPReach {
			return st, nil
		}
		time.Sleep(400 * time.Millisecond)
	}
	st = StatusInfo()
	if !st.HTTPReach {
		return st, fmt.Errorf("registry started but /v2/ not reachable on %s", Addr())
	}
	return st, nil
}

func Stop() error {
	eng := dockerBin()
	if eng == "" {
		return fmt.Errorf("docker/podman not found")
	}
	out, err := exec.Command(eng, "rm", "-f", containerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// EnsureCrane downloads crane to /usr/local/bin if missing.
func EnsureCrane() (string, error) {
	if p := CranePath(); p != "" {
		return p, nil
	}
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goarch == "aarch64" {
		goarch = "arm64"
	}
	// official go-containerregistry releases
	ver := "v0.20.2"
	name := fmt.Sprintf("go-containerregistry_%s_%s.tar.gz", goos, goarch)
	url := fmt.Sprintf("https://github.com/google/go-containerregistry/releases/download/%s/%s", ver, name)
	tmp := filepath.Join(os.TempDir(), name)
	cmd := exec.Command("curl", "-fsSL", "-o", tmp, url)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("download crane: %w: %s", err, strings.TrimSpace(string(out)))
	}
	destDir := "/usr/local/bin"
	_ = os.MkdirAll(destDir, 0o755)
	extract := exec.Command("tar", "-xzf", tmp, "-C", destDir, "crane")
	if out, err := extract.CombinedOutput(); err != nil {
		return "", fmt.Errorf("extract crane: %w: %s", err, strings.TrimSpace(string(out)))
	}
	_ = os.Chmod(filepath.Join(destDir, "crane"), 0o755)
	_ = os.Remove(tmp)
	p := CranePath()
	if p == "" {
		return "", fmt.Errorf("crane not found after install")
	}
	return p, nil
}

// Catalog lists repositories via registry HTTP API.
func Catalog() ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + Addr() + "/v2/_catalog")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("catalog HTTP %d", resp.StatusCode)
	}
	var body struct {
		Repositories []string `json:"repositories"`
	}
	if err := jsonDecode(resp, &body); err != nil {
		return nil, err
	}
	return body.Repositories, nil
}

func jsonDecode(resp *http.Response, v any) error {
	return json.NewDecoder(resp.Body).Decode(v)
}
