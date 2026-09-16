package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
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

// SelfReplace restarts unit after replacing binary (best-effort).
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

// VersionCompare returns true if remote is newer than local (string compare on tags).
func Newer(remote, local string) bool {
	remote = strings.TrimPrefix(remote, "v")
	local = strings.TrimPrefix(local, "v")
	return remote != "" && local != "" && remote != local && remote > local
}
