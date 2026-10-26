package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/upstream"
)

// PinnedSingBox is downloaded into offline cache (avoids GH API on OpenWrt).
const PinnedSingBox = upstream.SingBoxPinnedVersion

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
	ver := strings.TrimPrefix(PinnedSingBox, "v")
	name := fmt.Sprintf("sing-box-%s-linux-%s", ver, agentArch)
	dest := filepath.Join(dir, name)
	if st, err := os.Stat(dest); err == nil && st.Size() >= minSingBoxBytes {
		// Cache hit: still verify size; optional SHA file sidecar
		sumPath := dest + ".sha256"
		if b, err := os.ReadFile(sumPath); err == nil {
			want := strings.TrimSpace(string(b))
			got, herr := fileSHA256HexFile(dest)
			if herr != nil || !strings.EqualFold(got, want) {
				_ = os.Remove(dest)
				_ = os.Remove(sumPath)
			} else {
				return dest, nil
			}
		} else {
			return dest, nil // legacy cache without sidecar
		}
	}
	_ = os.Remove(dest) // drop partial/corrupt

	sbArch := singBoxArch(agentArch)
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
	if want := pinnedSingBoxSHA(sbArch); want != "" {
		got, err := fileSHA256HexFile(tgz)
		if err != nil {
			return "", err
		}
		if !matchSHA256(got, want) {
			return "", fmt.Errorf("sing-box tarball SHA mismatch arch=%s got=%s want=%s", sbArch, got, want)
		}
	} else if want := strings.TrimSpace(os.Getenv("NETDUCTOR_SINGBOX_SHA256")); want != "" {
		got, err := fileSHA256HexFile(tgz)
		if err != nil {
			return "", err
		}
		if !matchSHA256(got, want) {
			return "", fmt.Errorf("sing-box tarball SHA mismatch got=%s want=%s", got, want)
		}
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
	// Atomic publish: unique temp in same dir then rename
	data, err := os.ReadFile(bin)
	if err != nil {
		return "", err
	}
	tf, err := os.CreateTemp(filepath.Dir(dest), ".sb-pub-*")
	if err != nil {
		return "", err
	}
	tmpDest := tf.Name()
	if _, err := tf.Write(data); err != nil {
		tf.Close()
		_ = os.Remove(tmpDest)
		return "", err
	}
	if err := tf.Close(); err != nil {
		_ = os.Remove(tmpDest)
		return "", err
	}
	_ = os.Chmod(tmpDest, 0o755)
	if err := os.Rename(tmpDest, dest); err != nil {
		_ = os.Remove(tmpDest)
		return "", err
	}
	if sum, err := fileSHA256HexFile(dest); err == nil {
		_ = os.WriteFile(dest+".sha256", []byte(sum+"\n"), 0o644)
	}
	return dest, nil
}

func fileSHA256HexFile(path string) (string, error) {
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
