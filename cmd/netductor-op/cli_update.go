package main

import (
	"fmt"
	"os"
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
			fmt.Println("Downloads netductor-op-<goos>-<goarch> from GitHub Releases into this binary's directory.")
			return
		default:
			if !strings.HasPrefix(a, "-") && ver == "" {
				ver = a
			}
		}
	}
	if skip {
		_ = os.Setenv("NETDUCTOR_UPDATE_SKIP_VERIFY", "1")
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
	dest := exe
	// Prefer stable path if installed as netductor-op
	if base := filepath.Base(exe); base != "netductor-op" && base != "netductor" {
		dest = filepath.Join(filepath.Dir(exe), "netductor-op")
	}
	fmt.Fprintf(os.Stderr, "==> update operator %s/%s %s → %s\n", runtime.GOOS, runtime.GOARCH, tag, dest)
	// DownloadReleaseAsset component "op" — need assetName support
	if err := ndupdate.DownloadReleaseAsset(tag, "op", dest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("updated", dest, tag)
	fmt.Println("re-run: netductor-op version")
}
