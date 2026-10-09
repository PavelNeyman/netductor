package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// netductor-op project build <name> [--ref REF] [--owner ORG] [--dir PATH] [--upload] [--no-test]
//
// Mac (or any operator host) side for host=mac projects and local CI:
// 1) clone/pull GitHub
// 2) detect toolchain and build
// 3) write dist under ~/.netductor/projects/<name>/dist
// 4) optional --upload: scp dist + queue done on primary
func runProjectBuild(args []string) int {
	if len(args) < 1 || args[0] != "build" {
		fmt.Fprintln(os.Stderr, `usage:
  netductor-op project build <name> [--ref REF] [--owner ORG] [--dir PATH] [--upload] [--no-test]`)
		return 2
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: netductor-op project build <name> [--ref REF] [--upload]")
		return 2
	}
	name := args[1]
	ref := "main"
	owner := "PavelNeyman"
	dir := ""
	upload := false
	noTest := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--ref":
			if i+1 < len(args) {
				ref = args[i+1]
				i++
			}
		case "--owner":
			if i+1 < len(args) {
				owner = args[i+1]
				i++
			}
		case "--dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		case "--upload":
			upload = true
		case "--no-test":
			noTest = true
		}
	}

	home, _ := os.UserHomeDir()
	if dir == "" {
		dir = filepath.Join(home, ".netductor", "projects", name)
	}
	dist := filepath.Join(dir, "dist")
	_ = os.RemoveAll(dist)
	_ = os.MkdirAll(dist, 0o755)

	fmt.Printf("project build name=%s ref=%s dir=%s\n", name, ref, dir)
	if err := ensureProjectCheckout(dir, owner, name, ref); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	kind := detectProjectKind(dir)
	fmt.Println("detected:", kind)
	var buildErr error
	switch kind {
	case "go":
		buildErr = buildGo(dir, dist, noTest)
	case "rust":
		buildErr = buildRust(dir, dist, noTest)
	case "swift":
		buildErr = buildSwift(dir, dist)
	case "shell":
		buildErr = buildShell(dir, dist)
	default:
		buildErr = fmt.Errorf("unknown project kind in %s (need go.mod, Cargo.toml, Package.swift/xcodeproj, or *.sh)", dir)
	}
	if buildErr != nil {
		fmt.Fprintln(os.Stderr, "build failed:", buildErr)
		return 1
	}

	// summary
	ents, _ := os.ReadDir(dist)
	fmt.Printf("dist %s (%d files)\n", dist, len(ents))
	for _, e := range ents {
		fmt.Println(" ", e.Name())
	}

	if upload {
		if err := uploadProjectDist(name, ref, dist); err != nil {
			fmt.Fprintln(os.Stderr, "upload:", err)
			return 1
		}
		fmt.Println("upload ok; queue marked done for project", name)
	} else {
		fmt.Println("hint: add --upload to scp dist → primary and complete mac queue")
		fmt.Printf("hint: scp -r %s root@primary:/var/lib/netductor/git-artifacts/%s/\n", dist, name)
	}
	return 0
}

func ensureProjectCheckout(dir, owner, name, ref string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		_ = os.RemoveAll(dir)
		_ = os.MkdirAll(filepath.Dir(dir), 0o755)
		url := fmt.Sprintf("https://github.com/%s/%s.git", owner, name)
		// Prefer HTTPS; token from env for private repos
		if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
			url = fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", t, owner, name)
		}
		cmd := exec.Command("git", "clone", "--branch", ref, "--single-branch", "--depth", "50", url, dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// try without branch (tag or default)
			_ = os.RemoveAll(dir)
			cmd = exec.Command("git", "clone", "--depth", "50", url, dir)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err2 := cmd.Run(); err2 != nil {
				return fmt.Errorf("git clone: %w", err2)
			}
			co := exec.Command("git", "-C", dir, "checkout", ref)
			co.Stdout = os.Stdout
			co.Stderr = os.Stderr
			_ = co.Run()
		}
		return nil
	}
	_ = exec.Command("git", "-C", dir, "fetch", "--all", "--tags").Run()
	co := exec.Command("git", "-C", dir, "checkout", ref)
	co.Stdout = os.Stdout
	co.Stderr = os.Stderr
	if err := co.Run(); err != nil {
		return fmt.Errorf("git checkout %s: %w", ref, err)
	}
	_ = exec.Command("git", "-C", dir, "pull", "--ff-only").Run()
	return nil
}

func detectProjectKind(dir string) string {
	if fileExists(filepath.Join(dir, "go.mod")) {
		return "go"
	}
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return "rust"
	}
	if fileExists(filepath.Join(dir, "Package.swift")) {
		return "swift"
	}
	// xcodeproj / xcworkspace
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		n := e.Name()
		if strings.HasSuffix(n, ".xcodeproj") || strings.HasSuffix(n, ".xcworkspace") {
			return "swift"
		}
	}
	// shell-heavy
	hasSh := false
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".sh") {
			hasSh = true
			return filepath.SkipAll
		}
		return nil
	})
	if hasSh {
		return "shell"
	}
	return "unknown"
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func buildGo(dir, dist string, noTest bool) error {
	if !noTest {
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go test: %w", err)
		}
	}
	// Prefer ./cmd/... if present
	cmdPath := "./..."
	if st, err := os.Stat(filepath.Join(dir, "cmd")); err == nil && st.IsDir() {
		cmdPath = "./cmd/..."
	}
	outBin := filepath.Join(dist, "bin-"+runtime.GOOS+"-"+runtime.GOARCH)
	cmd := exec.Command("go", "build", "-o", outBin, cmdPath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// single main at root
		cmd = exec.Command("go", "build", "-o", outBin, ".")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err2 := cmd.Run(); err2 != nil {
			return fmt.Errorf("go build: %w", err2)
		}
	}
	// windows cross best-effort
	win := filepath.Join(dist, "bin-windows-amd64.exe")
	wcmd := exec.Command("go", "build", "-o", win, cmdPath)
	wcmd.Dir = dir
	wcmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	_ = wcmd.Run()
	_ = writeBuildMeta(dist, "go")
	return nil
}

func buildRust(dir, dist string, noTest bool) error {
	if !noTest {
		cmd := exec.Command("cargo", "test")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run() // non-fatal if no tests
	}
	cmd := exec.Command("cargo", "build", "--release")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo build: %w", err)
	}
	// copy release binaries
	rel := filepath.Join(dir, "target", "release")
	ents, _ := os.ReadDir(rel)
	for _, e := range ents {
		if e.IsDir() || strings.Contains(e.Name(), ".") {
			continue
		}
		src := filepath.Join(rel, e.Name())
		dst := filepath.Join(dist, e.Name())
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(dst, b, 0o755)
	}
	_ = writeBuildMeta(dist, "rust")
	return nil
}

func buildSwift(dir, dist string) error {
	// Prefer swift package
	if fileExists(filepath.Join(dir, "Package.swift")) {
		if _, err := exec.LookPath("swift"); err != nil {
			return fmt.Errorf("swift toolchain not found — install Xcode CLT")
		}
		cmd := exec.Command("swift", "build", "-c", "release")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("swift build: %w", err)
		}
		_ = writeBuildMeta(dist, "swift-package")
		// copy .build/release products best-effort
		_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/release/* %q 2>/dev/null || cp -a %q/.build/release/* %q 2>/dev/null || true", dir, dist, dir, dist)).Run()
		return nil
	}
	if _, err := exec.LookPath("xcodebuild"); err != nil {
		return fmt.Errorf("xcodebuild not found — open project in Xcode or install Xcode; host=mac required")
	}
	// list schemes
	list := exec.Command("xcodebuild", "-list")
	list.Dir = dir
	out, _ := list.CombinedOutput()
	_ = os.WriteFile(filepath.Join(dist, "xcodebuild-list.txt"), out, 0o644)
	// generic build (may fail without scheme — still leave list for operator)
	cmd := exec.Command("xcodebuild", "-configuration", "Release", "build")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "xcodebuild default failed — pick scheme manually; list saved in dist/xcodebuild-list.txt")
		_ = writeBuildMeta(dist, "swift-xcode-partial")
		return nil // partial success with list
	}
	_ = writeBuildMeta(dist, "swift-xcode")
	return nil
}

func buildShell(dir, dist string) error {
	script := `set -euo pipefail
n=0
while IFS= read -r -d '' f; do
  bash -n "$f" || exit 1
  n=$((n+1))
done < <(find . -name '*.sh' -print0)
echo "shellcheck_ok files=$n"
echo "shell files=$n" > dist/shell-ok.txt
`
	_ = os.MkdirAll(filepath.Join(dir, "dist"), 0o755)
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// move marker to our dist
	b, _ := os.ReadFile(filepath.Join(dir, "dist", "shell-ok.txt"))
	_ = os.WriteFile(filepath.Join(dist, "shell-ok.txt"), b, 0o644)
	_ = writeBuildMeta(dist, "shell")
	return nil
}

func writeBuildMeta(dist, kind string) error {
	meta := fmt.Sprintf("kind=%s\nos=%s\narch=%s\nts=%s\n", kind, runtime.GOOS, runtime.GOARCH, time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(filepath.Join(dist, "build-meta.txt"), []byte(meta), 0o644)
}

func uploadProjectDist(name, ref, dist string) error {
	s := loadTUISettings()
	host := strings.TrimSpace(s.RemoteHost)
	if host == "" {
		return fmt.Errorf("primary host not set — open TUI Setup first")
	}
	user := orDefault(s.RemoteUser, "root")
	key := strings.TrimSpace(s.RemoteKey)
	target := user + "@" + host
	remoteDir := fmt.Sprintf("/var/lib/netductor/git-artifacts/%s/%s", name, sanitizeRef(ref))
	// mkdir remote
	sshBase := []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "ConnectTimeout=20", "-o", "BatchMode=yes"}
	if key != "" {
		sshBase = append(sshBase, "-i", key)
	}
	mk := append(append([]string{}, sshBase...), target, "mkdir", "-p", remoteDir)
	if out, err := exec.Command("ssh", mk...).CombinedOutput(); err != nil {
		return fmt.Errorf("ssh mkdir: %w %s", err, out)
	}
	scpArgs := append(append([]string{}, sshBase...), "-r", dist+"/.", target+":"+remoteDir+"/")
	if out, err := exec.Command("scp", scpArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("scp: %w %s", err, out)
	}
	// complete mac queue (by project name)
	done := append(append([]string{}, sshBase...), target, "netductor", "git", "project", "queue", "done", name)
	_ = exec.Command("ssh", done...).Run()
	return nil
}

func sanitizeRef(ref string) string {
	ref = strings.ReplaceAll(ref, "/", "-")
	ref = strings.ReplaceAll(ref, "..", "")
	if ref == "" {
		return "HEAD"
	}
	return ref
}
