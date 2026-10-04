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

func githubToken() string {
	for _, k := range []string{"NETDUCTOR_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if t := strings.TrimSpace(os.Getenv(k)); t != "" {
			return t
		}
	}
	for _, p := range []string{"/etc/netductor/secrets/github_token", "/etc/netductor/github_token"} {
		if b, err := os.ReadFile(p); err == nil {
			if t := strings.TrimSpace(string(b)); t != "" {
				return t
			}
		}
	}
	return ""
}

func setGitHubHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "netductor-update")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if tok := githubToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

func releaseCachePath() string {
	return filepath.Join("/var/lib/netductor", "update-releases-cache.json")
}

type releaseCache struct {
	At       int64         `json:"at"`
	Latest   string        `json:"latest"`
	Releases []ReleaseInfo `json:"releases,omitempty"`
}

func loadReleaseCache(maxAge time.Duration) (releaseCache, bool) {
	var c releaseCache
	b, err := os.ReadFile(releaseCachePath())
	if err != nil {
		return c, false
	}
	if json.Unmarshal(b, &c) != nil || c.At == 0 {
		return c, false
	}
	if time.Since(time.Unix(c.At, 0)) > maxAge {
		return c, false
	}
	return c, true
}

func saveReleaseCache(latest string, list []ReleaseInfo) {
	_ = os.MkdirAll("/var/lib/netductor", 0o755)
	c := releaseCache{At: time.Now().Unix(), Latest: latest, Releases: list}
	raw, _ := json.Marshal(c)
	_ = os.WriteFile(releaseCachePath(), raw, 0o644)
}

// InvalidateReleaseCache drops on-disk release cache and in-process status cache.
// Call on explicit user Refresh in TG / Web / CLI / API (?force=1).
func InvalidateReleaseCache() {
	_ = os.Remove(releaseCachePath())
	statusCache = Status{}
	statusCacheAt = time.Time{}
}

// LatestReleaseTagForce ignores caches and re-queries GitHub.
func LatestReleaseTagForce() (string, error) {
	InvalidateReleaseCache()
	return LatestReleaseTag()
}

// ListReleasesForce ignores caches and re-queries GitHub.
func ListReleasesForce(limit int) ([]ReleaseInfo, error) {
	InvalidateReleaseCache()
	return ListReleases(limit)
}

// CheckStatusForce re-queries GitHub (no status/release cache).
func CheckStatusForce(local string) Status {
	InvalidateReleaseCache()
	return CheckStatus(local)
}

// latestViaRedirect uses github.com HTML redirect (no API quota).
func latestViaRedirect() (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest("GET", "https://github.com/"+Repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "netductor-update")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if i := strings.LastIndex(loc, "/tag/"); i >= 0 {
		tag := strings.TrimSpace(loc[i+5:])
		if q := strings.IndexAny(tag, "?#"); q >= 0 {
			tag = tag[:q]
		}
		if tag != "" {
			return tag, nil
		}
	}
	return "", fmt.Errorf("github redirect HTTP %d", resp.StatusCode)
}

// LatestReleaseTag from GitHub (no auth). Empty on error.
func LatestReleaseTag() (string, error) {
	if c, ok := loadReleaseCache(30 * time.Minute); ok && c.Latest != "" {
		return c.Latest, nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+Repo+"/releases/latest", nil)
	setGitHubHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		if tag, e2 := latestViaRedirect(); e2 == nil {
			saveReleaseCache(tag, nil)
			return tag, nil
		}
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		if tag, e2 := latestViaRedirect(); e2 == nil {
			saveReleaseCache(tag, nil)
			return tag, nil
		}
		if c, ok := loadReleaseCache(24 * time.Hour); ok && c.Latest != "" {
			return c.Latest, nil
		}
		return "", fmt.Errorf("github HTTP %d (rate limit — NETDUCTOR_GITHUB_TOKEN or secrets/github_token)", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		if tag, e2 := latestViaRedirect(); e2 == nil {
			saveReleaseCache(tag, nil)
			return tag, nil
		}
		return "", fmt.Errorf("github HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	tag := strings.TrimSpace(body.TagName)
	if tag != "" {
		saveReleaseCache(tag, nil)
	}
	return tag, nil
}

// ReleaseInfo is a GitHub release list entry (tag + optional name/date).
type ReleaseInfo struct {
	Tag         string `json:"tag"`
	Name        string `json:"name,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	Prerelease  bool   `json:"prerelease,omitempty"`
}

// ListReleases returns up to limit recent releases (newest first). limit<=0 → 15.
func ListReleases(limit int) ([]ReleaseInfo, error) {
	if limit <= 0 {
		limit = 15
	}
	if limit > 50 {
		limit = 50
	}
	if c, ok := loadReleaseCache(30 * time.Minute); ok && len(c.Releases) > 0 {
		if len(c.Releases) > limit {
			return c.Releases[:limit], nil
		}
		return c.Releases, nil
	}
	client := &http.Client{Timeout: 20 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", Repo, limit)
	req, _ := http.NewRequest("GET", url, nil)
	setGitHubHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return listReleasesFallback(limit)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		if list, e2 := listReleasesFallback(limit); e2 == nil {
			return list, nil
		}
		return nil, fmt.Errorf("github HTTP %d (rate limit — NETDUCTOR_GITHUB_TOKEN)", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		if list, e2 := listReleasesFallback(limit); e2 == nil {
			return list, nil
		}
		return nil, fmt.Errorf("github HTTP %d", resp.StatusCode)
	}
	var body []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		PublishedAt string `json:"published_at"`
		Prerelease  bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]ReleaseInfo, 0, len(body))
	for _, r := range body {
		tag := strings.TrimSpace(r.TagName)
		if tag == "" {
			continue
		}
		out = append(out, ReleaseInfo{
			Tag: tag, Name: r.Name, PublishedAt: r.PublishedAt, Prerelease: r.Prerelease,
		})
	}
	latest := ""
	if len(out) > 0 {
		latest = out[0].Tag
	}
	saveReleaseCache(latest, out)
	return out, nil
}

func listReleasesFallback(limit int) ([]ReleaseInfo, error) {
	if c, ok := loadReleaseCache(24 * time.Hour); ok && len(c.Releases) > 0 {
		if len(c.Releases) > limit {
			return c.Releases[:limit], nil
		}
		return c.Releases, nil
	}
	tag, err := latestViaRedirect()
	if err != nil {
		return nil, err
	}
	list := []ReleaseInfo{{Tag: tag}}
	saveReleaseCache(tag, list)
	return list, nil
}

// Status compares local version string to GitHub latest.
type Status struct {
	Local  string `json:"local"`
	Latest string `json:"latest,omitempty"`
	Update bool   `json:"update_available"`
	Error  string `json:"error,omitempty"`
}

func CheckStatus(local string) Status {
	st := Status{Local: strings.TrimSpace(local)}
	tag, err := LatestReleaseTag()
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Latest = tag
	st.Update = Newer(tag, st.Local)
	return st
}

func assetName(component string) string {
	goos, arch := runtime.GOOS, runtime.GOARCH
	switch component {
	case "agent":
		return fmt.Sprintf("netductor-agent-%s-%s", goos, arch)
	case "tg":
		return fmt.Sprintf("netductor-tg-%s-%s", goos, arch)
	case "op", "operator":
		return fmt.Sprintf("netductor-op-%s-%s", goos, arch)
	default: // node
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
			return fmt.Errorf("SHA256SUMS required (%v); set NETDUCTOR_UPDATE_SKIP_VERIFY=1 to bypass", err)
		}
		want, ok := sums[name]
		if !ok || want == "" {
			_ = os.Remove(tmp)
			return fmt.Errorf("no checksum entry for %s in SHA256SUMS", name)
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

// ApplyTag downloads a specific release tag asset to destPath (atomic .new + rename).
func ApplyTag(tag, component, destPath string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("empty tag")
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return DownloadReleaseAsset(tag, component, destPath)
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

var (
	statusCache    Status
	statusCacheAt  time.Time
	statusCacheTTL = 30 * time.Minute
)

// CheckStatusCached is CheckStatus with a 30m process-local cache (TUI/header safe).
func CheckStatusCached(local string) Status {
	if time.Since(statusCacheAt) < statusCacheTTL && statusCache.Local != "" {
		// re-evaluate Update against current local string
		st := statusCache
		st.Local = strings.TrimSpace(local)
		if st.Latest != "" {
			st.Update = Newer(st.Latest, st.Local)
		}
		return st
	}
	st := CheckStatus(local)
	if st.Error == "" {
		statusCache = st
		statusCacheAt = time.Now()
	}
	return st
}
