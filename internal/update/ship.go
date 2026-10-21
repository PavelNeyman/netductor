package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ci"
	"github.com/PavelNeyman/netductor/internal/gha"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// ShipOpts controls test→release matrix pipeline for netductor itself.
type ShipOpts struct {
	Tag        string // empty → NextReleaseTag()
	SkipTest   bool
	SkipDarwin bool
}

// Ship runs CI tests (unless SkipTest), then matrix build into local releases store.
// Failed tests do not publish a release directory.
func Ship(o ShipOpts) (tag, dst, logPath string, err error) {
	tag = strings.TrimSpace(o.Tag)
	if tag == "" {
		tag, err = NextReleaseTag()
		if err != nil {
			return "", "", "", err
		}
	}
	tag, err = ValidReleaseTag(tag)
	if err != nil {
		return "", "", "", err
	}
	ver := strings.TrimPrefix(tag, "v")

	work := filepath.Join(paths.StateDir(), "ci", "ship-work")
	_ = os.RemoveAll(work)
	if err := os.MkdirAll(work, 0o755); err != nil {
		return tag, "", "", err
	}
	defer os.RemoveAll(work)

	gitDir := filepath.Join(paths.StateDir(), "git", "netductor.git")
	if _, err := os.Stat(gitDir); err != nil {
		gitDir = filepath.Join(paths.StateDir(), "git", "netductor")
	}
	var logb strings.Builder
	fmt.Fprintf(&logb, "ship tag=%s skip_test=%v gitDir=%s\n", tag, o.SkipTest, gitDir)

	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		// Prefer exact release tag so Ship builds the requested commit, not tip HEAD.
		checked := false
		for _, ref := range []string{tag, "refs/tags/" + tag} {
			cmd := exec.Command("git", "--git-dir="+gitDir, "--work-tree="+work, "checkout", "-f", ref)
			out, cerr := cmd.CombinedOutput()
			logb.Write(out)
			if cerr == nil {
				fmt.Fprintf(&logb, "checkout %s ok\n", ref)
				checked = true
				break
			}
			fmt.Fprintf(&logb, "checkout %s: %v\n", ref, cerr)
		}
		if !checked {
			return tag, "", "", fmt.Errorf("git checkout failed for release tag %s (no main/HEAD fallback)", tag)
		}
		if ents, _ := os.ReadDir(work); len(ents) == 0 {
			return tag, "", "", fmt.Errorf("git checkout produced empty worktree for %s", tag)
		}
	} else {
		cmd := exec.Command("git", "clone", "--depth", "1", "--branch", tag, "https://github.com/PavelNeyman/netductor.git", work)
		out, cerr := cmd.CombinedOutput()
		logb.Write(out)
		if cerr != nil {
			return tag, "", "", fmt.Errorf("clone tag %s: %w", tag, cerr)
		}
	}

	if !o.SkipTest {
		wfPath, _ := gha.ResolveProjectWorkflow(work, "netductor")
		if wfPath == "" {
			wfPath, _ = gha.FindDefault(work)
		}
		if wfPath != "" {
			data, rerr := os.ReadFile(wfPath)
			if rerr != nil {
				return tag, "", "", rerr
			}
			w, perr := gha.Parse(data)
			if perr != nil {
				return tag, "", "", perr
			}
			fmt.Fprintf(&logb, "==> test %s\n", wfPath)
			out, terr := gha.Run(w, work, []string{"CI=true", "GOPROXY=https://proxy.golang.org,direct", "CGO_ENABLED=0"})
			logb.WriteString(out)
			if terr != nil {
				notify.AlertOnce("release:ship-test-fail:"+tag, fmt.Sprintf("🔴 Ship tests failed — <code>%s</code> not published", tag))
				_ = gitstore.PruneArtifacts(20)
				logPath = writeShipLog(tag, logb.String())
				return tag, "", logPath, fmt.Errorf("tests: %w", terr)
			}
		} else {
			fmt.Fprintf(&logb, "==> test go test sequential\n")
			out, terr := ci.Exec(ci.ExecOpts{
				Image: ci.ImageForGo(), WorkDir: work,
				Script: "go test -p 1 -parallel 1 ./...",
				Env:    []string{"CI=true", "GOPROXY=https://proxy.golang.org,direct", "CGO_ENABLED=0"},
			})
			logb.WriteString(out)
			if terr != nil {
				notify.AlertOnce("release:ship-test-fail:"+tag, fmt.Sprintf("🔴 Ship tests failed — <code>%s</code> not published", tag))
				logPath = writeShipLog(tag, logb.String())
				return tag, "", logPath, fmt.Errorf("tests: %w", terr)
			}
		}
	}

	_ = os.WriteFile(filepath.Join(work, "VERSION"), []byte(ver+"\n"), 0o644)
	if b, err := os.ReadFile(filepath.Join(work, "internal/version/version.go")); err == nil {
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			if strings.Contains(line, "const Release") {
				lines[i] = fmt.Sprintf("const Release = %q", ver)
			}
		}
		_ = os.WriteFile(filepath.Join(work, "internal/version/version.go"), []byte(strings.Join(lines, "\n")), 0o644)
	}

	fmt.Fprintf(&logb, "==> matrix build\n")
	out, berr := ci.Exec(ci.ExecOpts{
		Image: ci.ImageForGo(), WorkDir: work,
		Script:  buildScript(ver, o.SkipDarwin),
		Env:     []string{"GOPROXY=https://proxy.golang.org,direct", "CGO_ENABLED=0"},
		Network: "bridge",
	})
	logb.WriteString(out)
	logPath = writeShipLog(tag, logb.String())
	if berr != nil {
		WriteReleaseBuildStatus("fail", tag, "ship build fail")
		notify.AlertOnce("release:build-fail:"+tag, fmt.Sprintf("🔴 Ship build <code>%s</code> failed", tag))
		_ = gitstore.PruneArtifacts(20)
		return tag, "", logPath, fmt.Errorf("build: %w", berr)
	}
	dst, ierr := ImportReleaseDir(tag, filepath.Join(work, "dist"))
	if ierr != nil {
		return tag, "", logPath, ierr
	}
	WriteReleaseBuildStatus("ok", tag, "ship ok")
	notify.AlertOnce("release:built:"+tag, fmt.Sprintf("✅ Ship <code>%s</code> → local store (tests+matrix)", tag))
	_ = gitstore.PruneArtifacts(20)
	return tag, dst, logPath, nil
}

func writeShipLog(tag, body string) string {
	p := filepath.Join(paths.StateDir(), "ci", "ship-"+tag+".log")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(body), 0o600)
	return p
}
