package gitstore

import (
	"fmt"
	"net/url"

	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultUpstream for netductor development remote.
const DefaultUpstream = "https://github.com/PavelNeyman/netductor.git"

// MirrorEnsure creates bare repo name.git if missing and sets remote origin to upstreamURL.
func MirrorEnsure(name, upstreamURL string) (string, error) {
	name = sanitize(name)
	if name == "" {
		return "", fmt.Errorf("empty name")
	}
	if upstreamURL == "" {
		upstreamURL = DefaultUpstream
	}
	if err := EnsureRoot(); err != nil {
		return "", err
	}
	dir := filepath.Join(Root(), name+".git")
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		if _, err := Init(name); err != nil && !strings.Contains(err.Error(), "already exists") {
			return "", err
		}
	}
	// set/update origin (token injected for private GH; not written to logs)
	url := withGitHubAuth(upstreamURL)
	_ = exec.Command("git", "-C", dir, "remote", "remove", "origin").Run()
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", url).CombinedOutput(); err != nil {
		_ = exec.Command("git", "-C", dir, "remote", "set-url", "origin", url).Run()
		_ = out
	}
	return dir, nil
}

// MirrorFetch fetches tags and heads from origin into bare repo.
func MirrorFetch(name string) (string, error) {
	dir, err := repoDir(name)
	if err != nil {
		// try ensure with default upstream
		if _, e2 := MirrorEnsure(name, DefaultUpstream); e2 != nil {
			return "", err
		}
		dir, err = repoDir(name)
		if err != nil {
			return "", err
		}
	}
	// refresh origin URL with current token (private repos / rotated PAT)
	if out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").CombinedOutput(); err == nil {
		cur := strings.TrimSpace(string(out))
		// strip existing userinfo then re-apply
		plain := cur
		if u, e := parseMaybeURL(cur); e == nil && u.User != nil {
			u.User = nil
			plain = u.String()
		}
		if auth := withGitHubAuth(plain); auth != cur {
			_ = exec.Command("git", "-C", dir, "remote", "set-url", "origin", auth).Run()
		}
	}
	cmd := exec.Command("git", "-C", dir, "fetch", "--tags", "--prune", "origin", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ListTags returns tag names (newest first when possible).
func ListTags(name string) ([]string, error) {
	dir, err := repoDir(name)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "-C", dir, "tag", "-l", "--sort=-v:refname")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	var tags []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// CheckoutTag extracts tag into workDir (git archive or clone --shared).
func CheckoutTag(name, tag, workDir string) error {
	dir, err := repoDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	// verify tag exists
	if err := exec.Command("git", "-C", dir, "rev-parse", "--verify", tag).Run(); err != nil {
		return fmt.Errorf("tag %s not in local mirror (run: netductor git mirror-fetch %s)", tag, name)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", tag, dir, workDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout %s: %w\n%s", tag, err, string(out))
	}
	return nil
}


func parseMaybeURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
