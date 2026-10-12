package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// NextReleaseTag returns vMAJOR.MINOR.PATCH+1 after the highest known version.
// Sources: local releases, /etc/netductor/VERSION, GitHub latest (best-effort).
func NextReleaseTag() (string, error) {
	best := ""
	consider := func(c string) {
		c = strings.TrimSpace(c)
		if c == "" {
			return
		}
		if !strings.HasPrefix(c, "v") {
			c = "v" + strings.TrimPrefix(c, "v")
		}
		if best == "" || Newer(c, best) {
			best = c
		}
	}
	if t := LocalLatestTag(); t != "" {
		consider(t)
	}
	for _, p := range []string{
		filepath.Join(paths.EtcDir(), "VERSION"),
		"/etc/netductor/VERSION",
	} {
		if b, err := os.ReadFile(p); err == nil {
			consider(string(b))
		}
	}
	if gh, err := latestFromGitHub(); err == nil {
		consider(gh)
	}
	if best == "" {
		best = "v0.0.0"
	}
	next, err := bumpPatch(strings.TrimPrefix(best, "v"))
	if err != nil {
		return "", err
	}
	return "v" + next, nil
}

func bumpPatch(ver string) (string, error) {
	ver = strings.TrimPrefix(strings.TrimSpace(ver), "v")
	parts := strings.Split(ver, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	patStr := parts[2]
	if i := strings.IndexAny(patStr, "-+"); i >= 0 {
		patStr = patStr[:i]
	}
	pat, err3 := strconv.Atoi(patStr)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", fmt.Errorf("cannot bump version %q", ver)
	}
	return fmt.Sprintf("%d.%d.%d", maj, min, pat+1), nil
}
