package channels

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Incident rotation limits (disk safety).
const (
	IncidentMaxAge     = 7 * 24 * time.Hour
	IncidentMaxCount   = 20
	IncidentMaxTotalMB = 200
	IncidentCooldownHours  = 2
	historyMaxBytes    = 5 << 20 // 5 MiB hard cap on channel-history.jsonl
)

// IncidentDir stores diagnostic tar.gz bundles.
func IncidentDir() string {
	return filepath.Join(paths.StateDir(), "incidents")
}

// CaptureIncident writes a bounded diagnostic archive for a channel alert.
// Returns path to .tar.gz or "" if skipped/failed.
func CaptureIncident(reason string) (string, error) {
	_ = os.MkdirAll(IncidentDir(), 0o700)
	_ = PruneIncidents()
	_ = pruneHistorySize(historyMaxBytes)

	host, _ := os.Hostname()
	if host == "" {
		host = "node"
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	base := fmt.Sprintf("nd-incident-%s-%s", host, ts)
	dir := filepath.Join(IncidentDir(), base)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	meta := fmt.Sprintf("reason=%s\nhost=%s\nts=%s\n", reason, host, ts)
	_ = os.WriteFile(filepath.Join(dir, "meta.txt"), []byte(meta), 0o600)

	// Channel snapshot
	if rep, err := json.MarshalIndent(Collect(), "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "channel-status.json"), rep, 0o600)
	}
	if pts, err := ReadHistory(time.Now().UTC().Add(-2*time.Hour), ""); err == nil {
		b, _ := json.MarshalIndent(pts, "", "  ")
		if len(b) > 512<<10 {
			b = b[:512<<10]
		}
		_ = os.WriteFile(filepath.Join(dir, "channel-history-tail.json"), b, 0o600)
	}
	for _, p := range []string{
		"/var/lib/netductor/svc-paths-health.json",
		"/var/lib/netductor/failover-desired.json",
	} {
		if b, err := os.ReadFile(p); err == nil {
			_ = os.WriteFile(filepath.Join(dir, filepath.Base(p)), b, 0o600)
		}
	}

	// Bounded shell dumps (no secrets)
	runCapture(dir, "wg-show.txt", "wg", "show")
	runCapture(dir, "ss-443.txt", "sh", "-c", "ss -tn state established '( sport = :443 or dport = :443 )' 2>/dev/null | head -80")
	runCapture(dir, "ss-wss.txt", "sh", "-c", "ss -tn '( sport = :8454 or sport = :8455 or dport = :8454 or dport = :8455 )' 2>/dev/null | head -40")
	runCapture(dir, "uptime.txt", "sh", "-c", "uptime; free -m 2>/dev/null | head -5")
	runCapture(dir, "journal-sing-box.txt", "journalctl", "-u", "sing-box", "-n", "80", "--no-pager", "-o", "short-iso")
	runCapture(dir, "journal-wss.txt", "sh", "-c", "journalctl -u 'nd-wss*' -n 40 --no-pager -o short-iso 2>/dev/null | tail -40")

	// Optional short download probe (2s timeout, small)
	runCapture(dir, "curl-egress.txt", "sh", "-c",
		`curl -m 3 -o /dev/null -w 'speed_download=%{speed_download}\nhttp_code=%{http_code}\n' -s https://speed.cloudflare.com/__down?bytes=2000000 2>&1 | head -5`)

	tarPath := filepath.Join(IncidentDir(), base+".tar.gz")
	if err := tarGzDir(dir, tarPath); err != nil {
		return "", err
	}
	_ = PruneIncidents()
	return tarPath, nil
}

func runCapture(dir, name string, arg0 string, args ...string) {
	cmd := exec.Command(arg0, args...)
	out, err := cmd.CombinedOutput()
	if len(out) > 256<<10 {
		out = out[:256<<10]
	}
	body := out
	if err != nil && len(body) == 0 {
		body = []byte(err.Error())
	}
	_ = os.WriteFile(filepath.Join(dir, name), body, 0o600)
}

func tarGzDir(srcDir, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	base := filepath.Base(srcDir)
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(filepath.Join(base, rel))
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		rf, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, rf)
		_ = rf.Close()
		return copyErr
	})
}

// PruneIncidents removes old/oversized incident archives.
func PruneIncidents() error {
	dir := IncidentDir()
	_ = os.MkdirAll(dir, 0o700)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type item struct {
		path string
		mod  time.Time
		size int64
	}
	cutoff := time.Now().Add(-IncidentMaxAge)
	var files []item
	for _, e := range ents {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			_ = os.RemoveAll(p)
			continue
		}
		if !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.ModTime().Before(cutoff) {
			_ = os.Remove(p)
			continue
		}
		files = append(files, item{path: p, mod: st.ModTime(), size: st.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	var total int64
	for i, f := range files {
		if i >= IncidentMaxCount {
			_ = os.Remove(f.path)
			continue
		}
		if total+f.size > int64(IncidentMaxTotalMB)<<20 {
			_ = os.Remove(f.path)
			continue
		}
		total += f.size
	}
	return nil
}

func pruneHistorySize(maxBytes int64) error {
	path := historyPath()
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if st.Size() <= maxBytes {
		return nil
	}
	for _, d := range []time.Duration{12 * time.Hour, 6 * time.Hour, 2 * time.Hour, time.Hour} {
		_ = pruneHistory(d)
		st, err = os.Stat(path)
		if err != nil || st.Size() <= maxBytes {
			return err
		}
	}
	return nil
}
