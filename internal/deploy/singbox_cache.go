package deploy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PinnedSingBox is downloaded into offline cache (avoids GH API on OpenWrt).
const PinnedSingBox = "1.11.7"

// minSingBoxBytes — refuse tiny/corrupt cache entries (real binary is several MB).
const minSingBoxBytes = 1 << 20 // 1 MiB

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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := "sing-box-linux-" + agentArch
	dest := filepath.Join(dir, name)
	if st, err := os.Stat(dest); err == nil && st.Size() >= minSingBoxBytes {
		return dest, nil
	}
	_ = os.Remove(dest) // drop partial/corrupt

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

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(url)
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
	const maxTGZ = 80 << 20
	_, err = io.Copy(f, io.LimitReader(resp.Body, maxTGZ+1))
	_ = f.Close()
	if err != nil {
		return "", err
	}
	if st, _ := os.Stat(tgz); st != nil && st.Size() > maxTGZ {
		return "", fmt.Errorf("sing-box archive too large")
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
	st, err := os.Stat(bin)
	if err != nil || st.Size() < minSingBoxBytes {
		return "", fmt.Errorf("sing-box binary missing or too small")
	}
	// Atomic publish: write to dest.tmp then rename
	tmpDest := dest + ".tmp"
	data, err := os.ReadFile(bin)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(tmpDest, data, 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmpDest, dest); err != nil {
		_ = os.Remove(tmpDest)
		return "", err
	}
	return dest, nil
}
