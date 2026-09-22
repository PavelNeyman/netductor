// Package registry — thin local OCI registry (distribution/registry:2) + crane helper.
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

func AuthDir() string {
	return filepath.Join(DataDir(), "auth")
}

func Addr() string {
	if v := os.Getenv("NETDUCTOR_REGISTRY_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:5000"
}

func HostPort() (host, port string) {
	h, p, err := net.SplitHostPort(Addr())
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

func AuthEnabled() bool {
	_, err := os.Stat(filepath.Join(AuthDir(), "htpasswd"))
	return err == nil
}

type Status struct {
	OK        bool   `json:"ok"`
	Engine    string `json:"engine,omitempty"`
	Container string `json:"container"`
	Running   bool   `json:"running"`
	Addr      string `json:"addr"`
	DataDir   string `json:"data_dir"`
	HTTPReach bool   `json:"http_reachable"`
	Auth      bool   `json:"auth"`
	CranePath string `json:"crane,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Error     string `json:"error,omitempty"`
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func StatusInfo() Status {
	s := Status{
		Container: containerName,
		Addr:      Addr(),
		DataDir:   DataDir(),
		CranePath: CranePath(),
		Auth:      AuthEnabled(),
	}
	eng := dockerBin()
	s.Engine = eng
	if eng == "" {
		s.Error = "docker/podman not found"
		s.Hint = "install docker.io or podman, then: netductor registry ensure"
		return s
	}
	out, err := exec.Command(eng, "inspect", "-f", "{{.State.Running}}", containerName).CombinedOutput()
	s.Running = err == nil && strings.TrimSpace(string(out)) == "true"
	if s.Running {
		resp, err := registryGET("/v2/")
		if err == nil {
			_ = resp.Body.Close()
			// 200 anonymous, 401 when auth required
			s.HTTPReach = resp.StatusCode == 200 || resp.StatusCode == 401
		}
	}
	s.OK = s.Running && s.HTTPReach
	if s.OK {
		s.Hint = fmt.Sprintf("crane → %s  auth=%v", Addr(), s.Auth)
	} else if eng != "" && !s.Running {
		s.Hint = "netductor registry ensure"
	}
	return s
}

func Ensure() (Status, error) {
	eng := dockerBin()
	if eng == "" {
		return StatusInfo(), fmt.Errorf("docker or podman required")
	}
	_ = os.MkdirAll(DataDir(), 0o700)
	host, port := HostPort()
	st := StatusInfo()
	if st.Running && st.HTTPReach {
		return st, nil
	}
	_ = exec.Command(eng, "rm", "-f", containerName).Run()
	_ = exec.Command(eng, "pull", imageName).Run()

	args := []string{
		"run", "-d", "--name", containerName, "--restart", "unless-stopped",
		"-p", host + ":" + port + ":5000",
		"-v", DataDir() + "/data:/var/lib/registry",
	}
	_ = os.MkdirAll(filepath.Join(DataDir(), "data"), 0o700)
	if AuthEnabled() {
		args = append(args,
			"-v", AuthDir()+":/auth:ro",
			"-e", "REGISTRY_AUTH=htpasswd",
			"-e", "REGISTRY_AUTH_HTPASSWD_REALM=Netductor Registry",
			"-e", "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd",
		)
	}
	args = append(args, imageName)
	out, err := exec.Command(eng, args...).CombinedOutput()
	if err != nil {
		return StatusInfo(), fmt.Errorf("run registry: %w: %s", err, strings.TrimSpace(string(out)))
	}
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

// SetAuth writes htpasswd (bcrypt via htpasswd or apache2-utils) and restarts registry.
func SetAuth(user, pass string) error {
	user = strings.TrimSpace(user)
	if user == "" || pass == "" {
		return fmt.Errorf("user and password required")
	}
	_ = os.MkdirAll(AuthDir(), 0o700)
	path := filepath.Join(AuthDir(), "htpasswd")
	if _, err := exec.LookPath("htpasswd"); err == nil {
		cmd := exec.Command("htpasswd", "-Bbn", user, pass)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("htpasswd: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return os.WriteFile(path, out, 0o600)
	}
	// fallback: docker run httpd to generate
	eng := dockerBin()
	if eng == "" {
		return fmt.Errorf("htpasswd not found; install apache2-utils or httpd-tools")
	}
	out, err := exec.Command(eng, "run", "--rm", "httpd:2", "htpasswd", "-Bbn", user, pass).CombinedOutput()
	if err != nil {
		return fmt.Errorf("htpasswd via docker: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return err
	}
	_ = Stop()
	_, err = Ensure()
	return err
}

func ClearAuth() error {
	_ = os.Remove(filepath.Join(AuthDir(), "htpasswd"))
	_ = Stop()
	_, err := Ensure()
	return err
}

func EnsureCrane() (string, error) {
	if p := CranePath(); p != "" {
		return p, nil
	}
	goos, goarch := runtime.GOOS, runtime.GOARCH
	switch goos {
	case "linux":
		goos = "Linux"
	case "darwin":
		goos = "Darwin"
	case "windows":
		goos = "Windows"
	}
	switch goarch {
	case "amd64", "x86_64":
		goarch = "x86_64"
	case "arm64", "aarch64":
		goarch = "arm64"
	}
	ver := "v0.22.1"
	name := fmt.Sprintf("go-containerregistry_%s_%s.tar.gz", goos, goarch)
	url := fmt.Sprintf("https://github.com/google/go-containerregistry/releases/download/%s/%s", ver, name)
	tmp := filepath.Join(os.TempDir(), name)
	if out, err := exec.Command("curl", "-fsSL", "-o", tmp, url).CombinedOutput(); err != nil {
		return "", fmt.Errorf("download crane: %w: %s", err, strings.TrimSpace(string(out)))
	}
	destDir := "/usr/local/bin"
	_ = os.MkdirAll(destDir, 0o755)
	if out, err := exec.Command("tar", "-xzf", tmp, "-C", destDir, "crane").CombinedOutput(); err != nil {
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

// Catalog lists repository names.

func registryGET(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, "http://"+Addr()+path, nil)
	if err != nil {
		return nil, err
	}
	if u := os.Getenv("NETDUCTOR_REGISTRY_USER"); u != "" {
		req.SetBasicAuth(u, os.Getenv("NETDUCTOR_REGISTRY_PASSWORD"))
	}
	return httpClient().Do(req)
}

func Catalog() ([]string, error) {
	detail, err := CatalogDetail()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(detail))
	for _, r := range detail {
		out = append(out, r.Name)
	}
	return out, nil
}

type RepoInfo struct {
	Name string   `json:"name"`
	Tags []string `json:"tags,omitempty"`
}

// CatalogDetail returns repos with tags (digest listing via tags list API).
func CatalogDetail() ([]RepoInfo, error) {
	resp, err := registryGET("/v2/_catalog")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("registry requires auth (catalog via crane login or disable auth for local)")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("catalog HTTP %d", resp.StatusCode)
	}
	var body struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	var out []RepoInfo
	for _, name := range body.Repositories {
		ri := RepoInfo{Name: name}
		tr, err := registryGET("/v2/" + name + "/tags/list")
		if err == nil {
			var tb struct {
				Tags []string `json:"tags"`
			}
			if tr.StatusCode == 200 {
				_ = json.NewDecoder(tr.Body).Decode(&tb)
				ri.Tags = tb.Tags
			}
			_ = tr.Body.Close()
		}
		out = append(out, ri)
	}
	return out, nil
}
