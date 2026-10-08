package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildHost builds release matrix on the current machine (operator Mac or CI host)
// without docker. Writes into LocalReleasesDir()/<tag>. Useful for offline agent packs.
func BuildHost(o BuildOpts) (string, error) {
	tag, err := ValidReleaseTag(o.Tag)
	if err != nil {
		return "", err
	}
	ver := strings.TrimPrefix(tag, "v")
	work := o.WorkDir
	if work == "" {
		work = filepath.Join(os.TempDir(), "nd-release-"+tag)
		_ = os.RemoveAll(work)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return "", err
	}
	src := strings.TrimSpace(o.SourceURL)
	if src == "" {
		src = "https://github.com/" + Repo + ".git"
	}
	if _, err := os.Stat(filepath.Join(work, "go.mod")); err != nil {
		if err := cloneTag(src, tag, work); err != nil {
			return "", err
		}
	}
	_ = os.WriteFile(filepath.Join(work, "VERSION"), []byte(ver+"\n"), 0o644)

	dist := filepath.Join(work, "dist")
	_ = os.MkdirAll(dist, 0o755)
	ld := fmt.Sprintf("-s -w -X main.version=%s", ver)
	type item struct {
		goos, goarch, out, pkg, extra string
	}
	items := []item{
		{"linux", "amd64", "netductor-linux-amd64", "./cmd/netductor", ""},
		{"linux", "amd64", "netductor-tg-linux-amd64", "./cmd/netductor-tg", ""},
		{"linux", "amd64", "netductor-agent-linux-amd64", "./cmd/netductor-agent", ""},
		{"linux", "arm64", "netductor-agent-linux-arm64", "./cmd/netductor-agent", ""},
		{"linux", "arm", "netductor-agent-linux-arm", "./cmd/netductor-agent", "GOARM=7"},
		{"linux", "mipsle", "netductor-agent-linux-mipsle", "./cmd/netductor-agent", "GOMIPS=softfloat"},
		{"linux", "riscv64", "netductor-agent-linux-riscv64", "./cmd/netductor-agent", ""},
		{"linux", "amd64", "netductor-op-linux-amd64", "./cmd/netductor-op", ""},
	}
	if !o.SkipDarwin {
		items = append(items,
			item{"darwin", "arm64", "netductor-op-darwin-arm64", "./cmd/netductor-op", ""},
			item{"darwin", "amd64", "netductor-op-darwin-amd64", "./cmd/netductor-op", ""},
		)
	} else if runtime.GOOS == "darwin" {
		// always build native op for this Mac when skipping full darwin matrix
		arch := runtime.GOARCH
		items = append(items, item{"darwin", arch, "netductor-op-darwin-" + arch, "./cmd/netductor-op", ""})
	}
	for _, it := range items {
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ld, "-o", filepath.Join(dist, it.out), it.pkg)
		cmd.Dir = work
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS="+it.goos,
			"GOARCH="+it.goarch,
			"GOPROXY="+envOr("GOPROXY", "https://proxy.golang.org,direct"),
		)
		if it.extra != "" {
			parts := strings.SplitN(it.extra, "=", 2)
			if len(parts) == 2 {
				cmd.Env = append(cmd.Env, parts[0]+"="+parts[1])
			}
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("build %s: %w\n%s", it.out, err, string(out))
		}
	}
	sum := exec.Command("bash", "-c", "sha256sum netductor-* > SHA256SUMS")
	sum.Dir = dist
	if out, err := sum.CombinedOutput(); err != nil {
		return "", fmt.Errorf("sha256sum: %w\n%s", err, string(out))
	}
	dst, err := ImportReleaseDir(tag, dist)
	if err != nil {
		return "", err
	}
	return dst, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
