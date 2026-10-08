package main

import (
	"fmt"
	"os"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func runRelease(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor release list-local
  netductor release import <tag> <dir>   # copy assets into /var/lib/netductor/releases/<tag>
  netductor release path [tag]           # print store root or tag dir
  netductor release build <tag> [--skip-darwin]  # docker golang → local store
  netductor release build-host <tag> [--skip-darwin]  # local go (op/Mac)
  netductor release prune [--keep N]  # keep newest N local tags (default 5)`)
		return 2
	}
	switch args[0] {
	case "list-local", "list":
		tags, err := ndupdate.ListLocalTags()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(tags) == 0 {
			fmt.Println("(empty) " + ndupdate.LocalReleasesDir())
			return 0
		}
		for _, t := range tags {
			fmt.Println(t)
		}
		return 0
	case "path":
		if len(args) >= 2 {
			fmt.Println(ndupdate.LocalTagDir(args[1]))
			return 0
		}
		fmt.Println(ndupdate.LocalReleasesDir())
		return 0
	case "import":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor release import <tag> <dir>")
			return 2
		}
		dst, err := ndupdate.ImportReleaseDir(args[1], args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("imported", strings.TrimSpace(args[1]), "→", dst)
		return 0
	case "prune":
		keep := 5
		for i := 1; i < len(args); i++ {
			if args[i] == "--keep" && i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &keep)
				i++
			}
		}
		removed, err := ndupdate.PruneLocal(keep)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(removed) == 0 {
			fmt.Println("nothing to prune")
			return 0
		}
		for _, t := range removed {
			fmt.Println("removed", t)
		}
		return 0
	case "build-host":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor release build-host <tag> [--skip-darwin]")
			return 2
		}
		skipD := false
		for _, a := range args[2:] {
			if a == "--skip-darwin" {
				skipD = true
			}
		}
		fmt.Fprintln(os.Stderr, "release build-host", args[1], "(local go toolchain)")
		dst, err := ndupdate.BuildHost(ndupdate.BuildOpts{Tag: args[1], SkipDarwin: skipD})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok", dst)
		return 0
	case "build":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor release build <tag> [--skip-darwin]")
			return 2
		}
		skipD := false
		for _, a := range args[2:] {
			if a == "--skip-darwin" {
				skipD = true
			}
		}
		fmt.Fprintln(os.Stderr, "release build", args[1], "(docker golang; may take several minutes)")
		dst, err := ndupdate.BuildLocal(ndupdate.BuildOpts{Tag: args[1], SkipDarwin: skipD})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok", dst)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown release subcommand:", args[0])
		return 2
	}
}
