package main

import (
	"fmt"
	"os"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func runRelease(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor release list-local
  netductor release next              # print next auto tag (patch +1)
  netductor release ship [tag] [--skip-test] [--skip-darwin]
       # full cycle: tests → matrix → /var/lib/netductor/releases/<tag>
       # tag empty → auto next
  netductor release build [tag] [--skip-darwin]  # matrix only (no tests); tag empty → auto next
  netductor release build-host <tag> [--skip-darwin]
  netductor release import <tag> <dir>
  netductor release path [tag]
  netductor release prune [--keep N]
  netductor release prune-artifacts [--keep N]  # CI logs under git-artifacts`)
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
	case "next":
		t, err := ndupdate.NextReleaseTag()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(t)
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
	case "prune-artifacts":
		keep := 20
		for i := 1; i < len(args); i++ {
			if args[i] == "--keep" && i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &keep)
				i++
			}
		}
		if err := gitstore.PruneArtifacts(keep); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok artifacts keep", keep, gitstore.ArtifactDir())
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
		tag := ""
		skipD := false
		for _, a := range args[1:] {
			if a == "--skip-darwin" {
				skipD = true
				continue
			}
			if strings.HasPrefix(a, "-") {
				continue
			}
			tag = a
		}
		if tag == "" {
			var err error
			tag, err = ndupdate.NextReleaseTag()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Fprintln(os.Stderr, "auto tag", tag)
		}
		fmt.Fprintln(os.Stderr, "release build", tag, "(matrix only; use 'release ship' for tests+matrix)")
		dst, err := ndupdate.BuildLocal(ndupdate.BuildOpts{Tag: tag, SkipDarwin: skipD})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok", dst)
		return 0
	case "ship":
		fmt.Fprintln(os.Stderr, "release ship removed — use: netductor project ship netductor [--tag vX]")
		// still runs matrix for transition:
		tag := ""
		skipT, skipD := false, false
		for _, a := range args[1:] {
			switch a {
			case "--skip-test":
				skipT = true
			case "--skip-darwin":
				skipD = true
			default:
				if !strings.HasPrefix(a, "-") {
					tag = a
				}
			}
		}
		if tag == "" {
			fmt.Fprintln(os.Stderr, "ship auto-tag…")
		} else {
			fmt.Fprintln(os.Stderr, "ship", tag)
		}
		got, dst, logPath, err := ndupdate.Ship(ndupdate.ShipOpts{Tag: tag, SkipTest: skipT, SkipDarwin: skipD})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if logPath != "" {
				fmt.Fprintln(os.Stderr, "log", logPath)
			}
			return 1
		}
		fmt.Println("ok", got, "→", dst)
		if logPath != "" {
			fmt.Println("log", logPath)
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown release subcommand:", args[0])
		return 2
	}
}
