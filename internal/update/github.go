package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const Repo = "PavelNeyman/netductor"

// LatestReleaseTag from GitHub (no auth). Empty on error.
func LatestReleaseTag() (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+Repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return strings.TrimSpace(body.TagName), nil
}

func assetName(component string) string {
	goos, arch := runtime.GOOS, runtime.GOARCH
	switch component {
	case "agent":
		return fmt.Sprintf("netductor-agent-%s-%s", goos, arch)
	case "tg":
		return fmt.Sprintf("netductor-tg-%s-%s", goos, arch)
	default:
		return fmt.Sprintf("netductor-%s-%s", goos, arch)
	}
}

func downloadURL(tag, name string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", Repo, tag, name)
}

// FetchSHA256SUMS returns map basename -> hex digest from release SHA256SUMS file.
func FetchSHA256SUMS(tag string) (map[string]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(downloadURL(tag, "SHA256SUMS"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SHA256SUMS HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "hash  filename" or "hash *filename"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		h, name := fields[0], fields[len(fields)-1]
		name = strings.TrimPrefix(name, "*")
		out[filepath.Base(name)] = strings.ToLower(h)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty SHA256SUMS")
	}
	return out, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DownloadReleaseAsset writes binary to destPath. Verifies against SHA256SUMS unless NETDUCTOR_UPDATE_SKIP_VERIFY=1.
func DownloadReleaseAsset(tag, component, destPath string) error {
	name := assetName(component)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(downloadURL(tag, name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: HTTP %d", name, resp.StatusCode)
	}
	tmp := destPath + ".new"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if os.Getenv("NETDUCTOR_UPDATE_SKIP_VERIFY") != "1" {
		sums, err := FetchSHA256SUMS(tag)
		if err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("checksum file: %w", err)
		}
		want, ok := sums[name]
		if !ok {
			_ = os.Remove(tmp)
			return fmt.Errorf("no checksum for %s in SHA256SUMS", name)
		}
		got, err := fileSHA256(tmp)
		if err != nil {
			_ = os.Remove(tmp)
			return err
		}
		if !strings.EqualFold(got, want) {
			_ = os.Remove(tmp)
			return fmt.Errorf("checksum mismatch for %s", name)
		}
	}
	return os.Rename(tmp, destPath)
}

// SelfReplace downloads latest release asset and optionally restarts systemd unit.
func SelfReplace(component, destPath, unit string) error {
	tag, err := LatestReleaseTag()
	if err != nil {
		return err
	}
	if err := DownloadReleaseAsset(tag, component, destPath); err != nil {
		return err
	}
	if unit != "" {
		_ = exec.Command("systemctl", "restart", unit).Start()
	}
	return nil
}

func verParts(s string) (int, int, int) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	var a, b, c int
	if len(parts) > 0 {
		a, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		b, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		c, _ = strconv.Atoi(parts[2])
	}
	return a, b, c
}

// Newer returns true if remote tag is numerically newer than local VERSION.
func Newer(remote, local string) bool {
	ra, rb, rc := verParts(remote)
	la, lb, lc := verParts(local)
	if ra != la {
		return ra > la
	}
	if rb != lb {
		return rb > lb
	}
	return rc > lc
}

// WriteVERSION updates /etc/netductor/VERSION best-effort.
func WriteVERSION(tag string) {
	tag = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if tag == "" {
		return
	}
	_ = os.MkdirAll("/etc/netductor", 0o755)
	_ = os.WriteFile("/etc/netductor/VERSION", []byte(tag+"\n"), 0o644)
}
