package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/ci"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// BuildOpts controls release build on primary (P1).
type BuildOpts struct {
	// Tag e.g. v0.9.279
	Tag string
	// SourceURL override (default https://github.com/PavelNeyman/netductor)
	SourceURL string
	// SkipDarwin skips op darwin builds (faster on VPS).
	SkipDarwin bool
	// WorkDir if set is reused; otherwise temp under StateDir/ci/release-build
	WorkDir string
}

// BuildLocal clones tag (or uses workdir), builds release matrix in docker golang, imports into LocalTagDir.
func BuildLocal(o BuildOpts) (string, error) {
	tag, err := ValidReleaseTag(o.Tag)
	if err != nil {
		return "", err
	}
	ver := strings.TrimPrefix(tag, "v")
	src := strings.TrimSpace(o.SourceURL)
	if src == "" {
		src = "https://github.com/" + Repo + ".git"
	}

	work := o.WorkDir
	cleanup := false
	if work == "" {
		work = filepath.Join(paths.StateDir(), "ci", "release-build", tag+"-"+time.Now().UTC().Format("150405"))
		cleanup = true
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return "", err
	}
	if cleanup {
		defer func() { _ = os.RemoveAll(work) }()
	}

	// Prefer existing checkout; else shallow clone tag
	if _, err := os.Stat(filepath.Join(work, "go.mod")); err != nil {
		if err := cloneTag(src, tag, work); err != nil {
			return "", err
		}
	}

	// Pin version in worktree for ldflags/embed consistency
	_ = os.WriteFile(filepath.Join(work, "VERSION"), []byte(ver+"\n"), 0o644)
	if b, err := os.ReadFile(filepath.Join(work, "internal/version/version.go")); err == nil {
		lines := strings.Split(string(b), "
")
		for i, line := range lines {
			if strings.Contains(line, "const Release") {
				lines[i] = fmt.Sprintf("const Release = %q", ver)
			}
		}
		_ = os.WriteFile(filepath.Join(work, "internal/version/version.go"), []byte(strings.Join(lines, "
")), 0o644)
	}

	script := buildScript(ver, o.SkipDarwin)
	log, err := ci.Exec(ci.ExecOpts{
		Image:   ci.ImageForGo(),
		WorkDir: work,
		Script:  script,
		Env:     []string{"GOPROXY=https://proxy.golang.org,direct", "CGO_ENABLED=0"},
		Network: "bridge",
	})
	logPath := filepath.Join(paths.StateDir(), "ci", "release-"+tag+".log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	_ = os.WriteFile(logPath, []byte(log), 0o600)
	if err != nil {
		notify.AlertOnce("release:build-fail:"+tag, fmt.Sprintf("🔴 Release build <code>%s</code> failed — see %s", tag, logPath))
		return logPath, fmt.Errorf("build: %w\n%s", err, trimLog(log, 4000))
	}

	dist := filepath.Join(work, "dist")
	dst, err := ImportReleaseDir(tag, dist)
	if err != nil {
		notify.AlertOnce("release:import-fail:"+tag, fmt.Sprintf("🔴 Release import <code>%s</code>: %v", tag, err))
		return logPath, err
	}
	notify.AlertOnce("release:built:"+tag, fmt.Sprintf("✅ Release <code>%s</code> built → <code>%s</code> (local store)", tag, dst))
	return dst, nil
}

func cloneTag(src, tag, work string) error {
	// empty dir required for clone
	ents, _ := os.ReadDir(work)
	if len(ents) > 0 {
		return fmt.Errorf("workdir not empty for clone: %s", work)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", tag, src, work)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s %s: %w\n%s", src, tag, err, string(out))
	}
	return nil
}

func buildScript(ver string, skipDarwin bool) string {
	var b strings.Builder
	b.WriteString("set -euo pipefail\n")
	b.WriteString("export CGO_ENABLED=0\n")
	b.WriteString("mkdir -p dist\n")
	b.WriteString(fmt.Sprintf("LDFLAGS='-s -w -X main.version=%s'\n", ver))
	b.WriteString("build() { GOOS=${1:-linux} GOARCH=${2:-amd64} go build -trimpath -ldflags \"$LDFLAGS\" -o \"dist/$3\" \"$4\"; }\n")
	b.WriteString("build linux amd64 netductor-linux-amd64 ./cmd/netductor\n")
	b.WriteString("build linux amd64 netductor-tg-linux-amd64 ./cmd/netductor-tg\n")
	b.WriteString("build linux amd64 netductor-agent-linux-amd64 ./cmd/netductor-agent\n")
	b.WriteString("build linux arm64 netductor-agent-linux-arm64 ./cmd/netductor-agent\n")
	b.WriteString("GOARM=7 build linux arm netductor-agent-linux-arm ./cmd/netductor-agent\n")
	b.WriteString("GOMIPS=softfloat build linux mipsle netductor-agent-linux-mipsle ./cmd/netductor-agent\n")
	b.WriteString("build linux riscv64 netductor-agent-linux-riscv64 ./cmd/netductor-agent\n")
	b.WriteString("build linux amd64 netductor-op-linux-amd64 ./cmd/netductor-op\n")
	if !skipDarwin {
		b.WriteString("build darwin arm64 netductor-op-darwin-arm64 ./cmd/netductor-op\n")
		b.WriteString("build darwin amd64 netductor-op-darwin-amd64 ./cmd/netductor-op\n")
	}
	b.WriteString("(cd dist && sha256sum netductor-* > SHA256SUMS)\n")
	b.WriteString("ls -la dist\n")
	return b.String()
}

func trimLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
