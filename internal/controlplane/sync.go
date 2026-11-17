package controlplane

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// InboxDir holds pulled .ndenc archives for control standby (not auto-applied).
func InboxDir() string {
	return filepath.Join(paths.StateDir(), "backups", "control-inbox")
}

// PullLatest downloads active control's latest encrypted backup into the inbox.
// Does not decrypt or extract — promote path is explicit restore-inbox / recover.
func PullLatest(baseURL, bearer string, client *http.Client) (localPath, remoteName, shaHex string, err error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	bearer = strings.TrimSpace(bearer)
	if baseURL == "" || bearer == "" {
		return "", "", "", fmt.Errorf("control sync: base URL and bearer token required")
	}
	if !strings.HasPrefix(baseURL, "https://") && !strings.HasPrefix(baseURL, "http://") {
		return "", "", "", fmt.Errorf("control sync: URL must be http(s)://")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	url := baseURL + "/api/control/backup/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", "", "", fmt.Errorf("control sync: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	name := resp.Header.Get("X-Netductor-Backup-Name")
	if name == "" {
		name = "control-sync.ndenc"
	}
	name = filepath.Base(name)
	if name == "." || name == "/" {
		name = "control-sync.ndenc"
	}
	wantSHA := strings.TrimSpace(resp.Header.Get("X-Netductor-Backup-SHA256"))
	if err := os.MkdirAll(InboxDir(), 0o700); err != nil {
		return "", "", "", err
	}
	tmp := filepath.Join(InboxDir(), "."+name+".partial")
	_ = os.Remove(tmp)
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", "", err
	}
	h := sha256.New()
	const maxArch = 512 << 20
	n, err := io.Copy(f, io.TeeReader(io.LimitReader(resp.Body, maxArch+1), h))
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return "", "", "", err
	}
	if n > maxArch {
		_ = os.Remove(tmp)
		return "", "", "", fmt.Errorf("control sync: archive exceeds %d bytes", maxArch)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if wantSHA != "" && !strings.EqualFold(wantSHA, got) {
		_ = os.Remove(tmp)
		return "", "", "", fmt.Errorf("control sync: sha256 mismatch want=%s got=%s", wantSHA, got)
	}
	final := filepath.Join(InboxDir(), name)
	_ = os.Remove(final)
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", "", "", err
	}
	// optional components header → sidecar
	if comp := strings.TrimSpace(resp.Header.Get("X-Netductor-Components")); comp != "" {
		_ = os.WriteFile(filepath.Join(InboxDir(), "COMPONENTS.txt"), []byte(strings.ReplaceAll(comp, ",", "\n")+"\n"), 0o644)
	}
	meta := fmt.Sprintf("name=%s\nsha256=%s\npulled_at=%s\nsource=%s\n", name, got, time.Now().UTC().Format(time.RFC3339), baseURL)
	_ = os.WriteFile(filepath.Join(InboxDir(), "LATEST.txt"), []byte(meta), 0o644)
	return final, name, got, nil
}

// LatestInbox returns newest .ndenc in inbox (by mtime).
func LatestInbox() (string, error) {
	ents, err := os.ReadDir(InboxDir())
	if err != nil {
		return "", err
	}
	var latest string
	var latestT time.Time
	for _, e := range ents {
		n := e.Name()
		if !strings.HasSuffix(n, ".ndenc") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if latest == "" || info.ModTime().After(latestT) {
			latestT = info.ModTime()
			latest = filepath.Join(InboxDir(), n)
		}
	}
	if latest == "" {
		return "", fmt.Errorf("control inbox empty — run: netductor control sync-pull")
	}
	return latest, nil
}

// SyncConfig from env (standby timer / CLI defaults).
func SyncURL() string {
	return strings.TrimSpace(firstNonEmpty(os.Getenv("NETDUCTOR_CONTROL_SYNC_URL"), os.Getenv("CONTROL_SYNC_URL")))
}

func SyncToken() string {
	if t := strings.TrimSpace(os.Getenv("NETDUCTOR_CONTROL_SYNC_TOKEN")); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("CONTROL_SYNC_TOKEN")); t != "" {
		return t
	}
	// secret file preferred on node
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", "control_standby_token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}
