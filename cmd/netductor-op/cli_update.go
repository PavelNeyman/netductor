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
			printUpdateHelp()
			return
		}
	}
	ver := ""
	skip := false
	viaBrew := false
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
		case "--via-brew":
			// Optional: force Homebrew formula path instead of GitHub self-update.
			viaBrew = true
		case "--help", "-h":
			printUpdateHelp()
			return
		default:
			if !strings.HasPrefix(a, "-") && ver == "" {
				ver = a
			}
		}
	}
	if viaBrew || os.Getenv("NETDUCTOR_OP_UPDATE_VIA_BREW") == "1" {
		runUpdateViaBrew()
		return
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

	dest, how, err := resolveOpUpdateDest()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "==> update operator %s/%s %s → %s (%s)\n", runtime.GOOS, runtime.GOARCH, tag, dest, how)
	if err := ndupdate.DownloadReleaseAsset(tag, "op", dest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

func printUpdateHelp() {
	fmt.Println("netductor-op update check|list [--refresh|--force]")
	fmt.Println("netductor-op update [--version X] [--skip-verify] [--via-brew]")
	fmt.Println("")
	fmt.Println("Model (cask-like):")
	fmt.Println("  • brew install/upgrade  — optional entry / first install")
	fmt.Println("  • netductor-op update   — day-to-day: GitHub Release + SHA256 (default)")
	fmt.Println("  • --via-brew            — only bump Homebrew formula (may lag tip)")
	fmt.Println("")
	fmt.Println("When installed via brew, self-update writes into the Cellar binary")
	fmt.Println("(resolved path). The bin/ symlink is never replaced with a plain file.")
}

// resolveOpUpdateDest picks where to write the new binary.
// Critical: never overwrite a Homebrew bin/ symlink — that leaves a plain file and breaks brew link.
func resolveOpUpdateDest() (dest, how string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	resolved := exe
	if r, e := filepath.EvalSymlinks(exe); e == nil && r != "" {
		resolved = r
	}
	resolved = filepath.Clean(resolved)
	exe = filepath.Clean(exe)

	prefix := brewPrefix()
	binLink := filepath.Join(prefix, "bin", "netductor-op")
	cellarRoot := filepath.Join(prefix, "Cellar", "netductor-op")

	// Broken state from older self-update: plain file in bin/ shadows Cellar.
	if fi, e := os.Lstat(binLink); e == nil && fi.Mode()&os.ModeSymlink == 0 && !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "note: %s is a plain file (old self-update broke brew link).\n", binLink)
		fmt.Fprintf(os.Stderr, "      Will update that file in place. Prefer after: brew link --overwrite netductor-op\n")
		return binLink, "brew-bin-plain-file", nil
	}

	// Healthy brew: bin is symlink → Cellar/.../bin/netductor-op
	if strings.HasPrefix(resolved, cellarRoot+string(os.PathSeparator)) {
		return resolved, "homebrew-cellar", nil
	}

	// Running via bin path that is a symlink — resolve already done; if still bin, prefer EvalSymlinks of bin
	if exe == binLink || resolved == binLink {
		if r, e := filepath.EvalSymlinks(binLink); e == nil && r != "" && r != binLink {
			return filepath.Clean(r), "homebrew-cellar-via-bin", nil
		}
	}

	dest = resolved
	if base := filepath.Base(dest); base != "netductor-op" && base != "netductor" {
		dest = filepath.Join(filepath.Dir(dest), "netductor-op")
	}
	return dest, "direct", nil
}

func runUpdateViaBrew() {
	if _, err := exec.LookPath("brew"); err != nil {
		fmt.Fprintln(os.Stderr, "brew not found; use plain: netductor-op update")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "==> brew update && brew upgrade netductor-op")
	_ = exec.Command("brew", "update").Run()
	cmd := exec.Command("brew", "upgrade", "netductor-op")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// link may fail if plain file in bin
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintf(os.Stderr, "If link failed:\n  rm -f %s/bin/netductor-op\n  brew link --overwrite netductor-op\n", brewPrefix())
		os.Exit(1)
	}
	fmt.Println("re-run: hash -r && netductor-op version")
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
