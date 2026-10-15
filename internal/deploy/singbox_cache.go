package deploy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PinnedSingBox is downloaded into offline cache (avoids GH API on OpenWrt).
const PinnedSingBox = "1.11.7"

// SingBoxCacheDir returns ~/.cache/netductor/sing-box
func SingBoxCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "netductor", "sing-box")
}

// singBoxArch maps GOARCH/agent arch to sing-box release arch name.
func singBoxArch(agentArch string) string {
	switch agentArch {
	case "arm":
		return "armv7"
	case "mipsle":
		return "mipsle" // softfloat builds use mipsle in recent releases
	default:
		return agentArch
	}
}

// EnsureSingBoxCached downloads (if needed) sing-box for agentArch into cache dir; returns path.
func EnsureSingBoxCached(agentArch string) (string, error) {
	if agentArch == "" || agentArch == "auto" {
		return "", fmt.Errorf("sing-box cache: arch required")
	}
	dir := SingBoxCacheDir()
	_ = os.MkdirAll(dir, 0o755)
	name := "sing-box-linux-" + agentArch
	dest := filepath.Join(dir, name)
	if st, err := os.Stat(dest); err == nil && st.Size() > 1000 {
		return dest, nil
	}
	sbArch := singBoxArch(agentArch)
	ver := strings.TrimPrefix(PinnedSingBox, "v")
	url := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/sing-box-%s-linux-%s.tar.gz",
		ver, ver, sbArch)
	tmp, err := os.MkdirTemp("", "sb-cache-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	tgz := filepath.Join(tmp, "sb.tgz")
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("sing-box download HTTP %d %s", resp.StatusCode, url)
	}
	f, err := os.Create(tgz)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return "", err
	}
	if out, err := exec.Command("tar", "-xzf", tgz, "-C", tmp).CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar: %s %v", out, err)
	}
	var bin string
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "sing-box" {
			bin = path
		}
		return nil
	})
	if bin == "" {
		return "", fmt.Errorf("sing-box binary not in archive for %s", sbArch)
	}
	data, err := os.ReadFile(bin)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, data, 0o755); err != nil {
		return "", err
	}
	return dest, nil
}
