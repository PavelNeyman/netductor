package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

// DownloadReleaseAsset writes binary to destPath.
func DownloadReleaseAsset(tag, component, destPath string) error {
	name := assetName(component)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", Repo, tag, name)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: HTTP %d", name, resp.StatusCode)
	}
	f, err := os.OpenFile(destPath+".new", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return err
	}
	return os.Rename(destPath+".new", destPath)
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

// Parse soft semver for comparison (major.minor.patch, ignores -suffix).
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
