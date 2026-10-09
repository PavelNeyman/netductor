package update

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// LocalReleasesDir is /var/lib/netductor/releases (override NETDUCTOR_RELEASES_DIR).
func LocalReleasesDir() string {
	if d := strings.TrimSpace(os.Getenv("NETDUCTOR_RELEASES_DIR")); d != "" {
		return d
	}
	return filepath.Join(paths.StateDir(), "releases")
}

// LocalTagDir returns releases/<tag>/ (tag normalized with v prefix when possible).
func LocalTagDir(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag != "" && !strings.HasPrefix(tag, "v") {
		if t, err := ValidReleaseTag(tag); err == nil {
			tag = t
		} else {
			tag = "v" + strings.TrimPrefix(tag, "v")
		}
	}
	return filepath.Join(LocalReleasesDir(), tag)
}

// ListLocalTags returns tag directory names under the local store.
func ListLocalTags() ([]string, error) {
	root := LocalReleasesDir()
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() && strings.HasPrefix(e.Name(), "v") {
			out = append(out, e.Name())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] > out[j] })
	return out, nil
}

// TryLocalReleaseAsset copies component binary from local store when present.
func TryLocalReleaseAsset(tag, component, destPath string) (bool, error) {
	tag, err := ValidReleaseTag(tag)
	if err != nil {
		return false, err
	}
	return tryLocalNamedAsset(tag, assetName(component), destPath)
}

func tryLocalNamedAsset(tag, name, destPath string) (bool, error) {
	name = filepath.Base(name)
	dir := LocalTagDir(tag)
	src := filepath.Join(dir, name)
	if _, err := os.Stat(src); err != nil {
		return false, nil
	}
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	if b, err := os.ReadFile(sumsPath); err == nil {
		sums, err := ParseSHA256SUMS(b)
		if err != nil {
			return false, fmt.Errorf("local SHA256SUMS: %w", err)
		}
		want, ok := sums[name]
		if !ok {
			return false, fmt.Errorf("local SHA256SUMS: no entry for %s", name)
		}
		got, err := fileSHA256(src)
		if err != nil {
			return false, err
		}
		if !strings.EqualFold(got, want) {
			return false, fmt.Errorf("local %s sha mismatch", name)
		}
	}
	in, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	tmp := destPath + ".tmp"
	if err := os.WriteFile(tmp, in, 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return false, err
	}
	_ = os.Chmod(destPath, 0o755)
	return true, nil
}

// ImportReleaseDir copies all files from srcDir into LocalTagDir(tag).
func ImportReleaseDir(tag, srcDir string) (string, error) {
	tag, err := ValidReleaseTag(tag)
	if err != nil {
		return "", err
	}
	dst := LocalTagDir(tag)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", err
	}
	ents, err := os.ReadDir(srcDir)
	if err != nil {
		return "", err
	}
	n := 0
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			return "", err
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(e.Name(), "netductor") {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, mode); err != nil {
			return "", err
		}
		n++
	}
	if n == 0 {
		return "", fmt.Errorf("no files in %s", srcDir)
	}
	return dst, nil
}

// PruneLocal keeps the newest keep tags (by semver-ish name sort reverse), deletes the rest.
func PruneLocal(keep int) (removed []string, err error) {
	if keep < 1 {
		keep = 3
	}
	tags, err := ListLocalTags()
	if err != nil {
		return nil, err
	}
	if len(tags) <= keep {
		return nil, nil
	}
	// tags already from ReadDir — sort reverse version-ish
	sort.Slice(tags, func(i, j int) bool {
		return tags[i] > tags[j]
	})
	for _, t := range tags[keep:] {
		dir := LocalTagDir(t)
		if err := os.RemoveAll(dir); err != nil {
			return removed, err
		}
		removed = append(removed, t)
	}
	return removed, nil
}

// MaybeAutoBuildNewest schedules a local release build when the newest mirror tag is not in the local store.
// Disable with NETDUCTOR_RELEASE_AUTO_BUILD=0.
func MaybeAutoBuildNewest() error {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_RELEASE_AUTO_BUILD")); v == "0" || strings.EqualFold(v, "false") {
		WriteReleaseBuildStatus("skipped", "", "auto-build disabled")
		return nil
	}
	tags, err := listMirrorTags("netductor")
	if err != nil || len(tags) == 0 {
		return err
	}
	newest := tags[0]
	local, _ := ListLocalTags()
	for _, t := range local {
		if t == newest || "v"+t == newest || t == strings.TrimPrefix(newest, "v") {
			WriteReleaseBuildStatus("uptodate", newest, "local store already has newest mirror tag")
			return nil
		}
	}
	WriteReleaseBuildStatus("scheduled", newest, "auto-build after mirror-fetch")
	_ = notify.Telegram(fmt.Sprintf("🔨 Auto release build scheduled: <code>%s</code>", newest))
	return ScheduleBuild(newest, true)
}

// ReleaseBuildStatus is written for metrics/doctor.
type ReleaseBuildStatus struct {
	State   string `json:"state"` // scheduled|uptodate|skipped|ok|fail
	Tag     string `json:"tag,omitempty"`
	Detail  string `json:"detail,omitempty"`
	At      string `json:"at"`
}

func releaseStatusPath() string {
	return filepath.Join(paths.StateDir(), "release-build-status.json")
}

func WriteReleaseBuildStatus(state, tag, detail string) {
	st := ReleaseBuildStatus{
		State: state, Tag: tag, Detail: detail,
		At: time.Now().UTC().Format(time.RFC3339),
	}
	_ = os.MkdirAll(filepath.Dir(releaseStatusPath()), 0o755)
	b, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(releaseStatusPath(), append(b, '\n'), 0o644)
}

func ReadReleaseBuildStatus() ReleaseBuildStatus {
	b, err := os.ReadFile(releaseStatusPath())
	if err != nil {
		return ReleaseBuildStatus{}
	}
	var st ReleaseBuildStatus
	_ = json.Unmarshal(b, &st)
	return st
}

func listMirrorTags(name string) ([]string, error) {
	dir := filepath.Join(paths.StateDir(), "git", name+".git")
	cmd := exec.Command("git", "-C", dir, "tag", "-l", "--sort=-v:refname")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "v") {
			tags = append(tags, line)
		}
	}
	return tags, nil
}
