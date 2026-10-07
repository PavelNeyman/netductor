package logs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// ExportHours writes a tar.gz of recent journals under ExportDir. Returns path.
func ExportHours(hours int) (string, error) {
	if hours < 1 {
		hours = 1
	}
	if hours > 48 {
		hours = 48
	}
	_ = os.MkdirAll(ExportDir(), 0o700)
	host, _ := os.Hostname()
	if host == "" {
		host = "node"
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	dir := filepath.Join(ExportDir(), fmt.Sprintf("nd-logs-%s-%dh-%s", host, hours, ts))
	_ = os.MkdirAll(dir, 0o700)

	since := fmt.Sprintf("%d hours ago", hours)
	units := []string{"sing-box", "netductor-api", "netductor-telegram-bot", "netductor-secondary-agent", "netductor-agent", "blocky"}
	for _, u := range units {
		out, _ := exec.Command("journalctl", "-u", u, "--since", since, "-o", "short-iso", "--no-pager").CombinedOutput()
		_ = os.WriteFile(filepath.Join(dir, "journal-"+u+".log"), out, 0o600)
	}
	// meta
	meta := fmt.Sprintf("host=%s hours=%d since=%s ts=%s\nversion=%s\n", host, hours, since, ts, readVersion())
	_ = os.WriteFile(filepath.Join(dir, "meta.txt"), []byte(meta), 0o600)

	tarPath := dir + ".tar.gz"
	cmd := exec.Command("tar", "-C", ExportDir(), "-czf", tarPath, filepath.Base(dir))
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar: %v %s", err, out)
	}
	_ = os.RemoveAll(dir)
	return tarPath, nil
}

func readVersion() string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
