package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func runOpUpdate(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "check", "status":
			force := false
			for _, a := range args[1:] {
				if a == "--refresh" || a == "--force" || a == "-f" {
					force = true
				}
			}
			var st ndupdate.Status
			if force {
				st = ndupdate.CheckStatusForce(version)
			} else {
				st = ndupdate.CheckStatus(version)
			}
			fmt.Printf("local=%s latest=%s update=%v %s\n", st.Local, st.Latest, st.Update, st.Error)
			return
		case "list", "releases":
			force := false
			for _, a := range args[1:] {
				if a == "--refresh" || a == "--force" || a == "-f" {
					force = true
				}
			}
			var list []ndupdate.ReleaseInfo
			var err error
			if force {
				list, err = ndupdate.ListReleasesForce(15)
			} else {
				list, err = ndupdate.ListReleases(15)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			for _, r := range list {
				fmt.Printf("%s  %s\n", r.Tag, r.PublishedAt)
			}
			return
		case "help", "--help", "-h":
			fmt.Println("netductor-op update check|list [--refresh|--force]")
			fmt.Println("netductor-op update [--version X] [--skip-verify]")
			fmt.Println("Homebrew installs: use brew upgrade netductor-op (self-update refuses brew paths).")
			return
		}
	}
	ver := ""
	skip := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--version", "-v":
			if i+1 < len(args) {
				i++
				ver = args[i]
			}
		case "--skip-verify":
			skip = true
		case "--help", "-h":
			fmt.Println("netductor-op update check|list")
			fmt.Println("netductor-op update [--version X] [--skip-verify]")
			fmt.Println("Downloads netductor-op-<goos>-<goarch> from GitHub Releases.")
			fmt.Println("If this binary is from Homebrew, self-update is disabled — use brew upgrade.")
			return
		default:
			if !strings.HasPrefix(a, "-") && ver == "" {
				ver = a
			}
		}
	}
	if skip {
		_ = os.Setenv("NETDUCTOR_UPDATE_SKIP_VERIFY", "1")
		_ = os.Setenv("NETDUCTOR_UPDATE_SKIP_VERIFY_CONFIRM", "yes")
	}
	tag := strings.TrimSpace(ver)
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			fmt.Fprintln(os.Stderr, "latest:", err)
			os.Exit(1)
		}
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Resolve symlinks (Homebrew bin → Cellar).
	resolved := exe
	if r, err := filepath.EvalSymlinks(exe); err == nil && r != "" {
		resolved = r
	}
	dest := resolved
	if base := filepath.Base(dest); base != "netductor-op" && base != "netductor" {
		dest = filepath.Join(filepath.Dir(dest), "netductor-op")
	}

	if brewManagedPath(exe, resolved) {
		prefix := brewPrefix()
		fmt.Fprintln(os.Stderr, "netductor-op is Homebrew-managed — refusing self-update into brew paths.")
		fmt.Fprintln(os.Stderr, "Self-update previously left a plain file in bin/ that shadows Cellar and breaks `brew link`.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Fix + upgrade:")
		fmt.Fprintf(os.Stderr, "  rm -f %s/bin/netductor-op\n", prefix)
		fmt.Fprintln(os.Stderr, "  brew update && brew upgrade netductor-op")
		fmt.Fprintln(os.Stderr, "  brew link --overwrite netductor-op")
		fmt.Fprintln(os.Stderr, "  hash -r && netductor-op version")
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr, "==> update operator %s/%s %s → %s\n", runtime.GOOS, runtime.GOARCH, tag, dest)
	if err := ndupdate.DownloadReleaseAsset(tag, "op", dest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Confirm on-disk version (avoid "updated" while PATH still hits another binary).
	got := opVersionOf(dest)
	fmt.Println("updated", dest, tag)
	if got != "" {
		fmt.Println("on-disk version:", got)
	}
	fmt.Println("re-run: hash -r && netductor-op version")
	which, _ := exec.LookPath("netductor-op")
	if which != "" && !sameFile(which, dest) {
		fmt.Fprintf(os.Stderr, "WARN: PATH netductor-op is %s (not the file just updated). hash -r or check PATH.\n", which)
	}
}

func brewPrefix() string {
	if p := strings.TrimSpace(os.Getenv("HOMEBREW_PREFIX")); p != "" {
		return p
	}
	if out, err := exec.Command("brew", "--prefix").Output(); err == nil {
		if p := strings.TrimSpace(string(out)); p != "" {
			return p
		}
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "/opt/homebrew"
	}
	return "/usr/local"
}

// brewManagedPath: Cellar install or a binary sitting in Homebrew's bin/ (incl. broken plain file).
func brewManagedPath(exe, resolved string) bool {
	p := brewPrefix()
	cellar := filepath.Join(p, "Cellar", "netductor-op")
	bin := filepath.Join(p, "bin", "netductor-op")
	for _, s := range []string{exe, resolved} {
		s = filepath.Clean(s)
		if strings.HasPrefix(s, cellar+string(os.PathSeparator)) || s == cellar {
			return true
		}
		if s == bin {
			return true
		}
		if strings.Contains(s, "/Cellar/netductor-op/") {
			return true
		}
	}
	// Symlink in bin pointing at Cellar
	if fi, err := os.Lstat(bin); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if t, err := os.Readlink(bin); err == nil && strings.Contains(t, "Cellar/netductor-op") {
			if filepath.Clean(exe) == bin || filepath.Clean(resolved) == filepath.Clean(filepath.Join(filepath.Dir(bin), t)) {
				return true
			}
		}
	}
	return false
}

func opVersionOf(path string) string {
	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sameFile(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil {
		ra = a
	}
	if err2 != nil {
		rb = b
	}
	ra, rb = filepath.Clean(ra), filepath.Clean(rb)
	if ra == rb {
		return true
	}
	fa, e1 := os.Stat(ra)
	fb, e2 := os.Stat(rb)
	if e1 != nil || e2 != nil {
		return false
	}
	return os.SameFile(fa, fb)
}
