package main

import (
	"fmt"
	"os"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

func runOpRelease(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor-op release list-local|tags|mirror-fetch|path [tag]  # via SSH primary
  netductor-op release build <tag> [--skip-darwin]               # via SSH primary (docker)
  netductor-op release build-host <tag> [--skip-darwin]          # local go → ~/.netductor/releases
  netductor-op release prune [--keep N]                          # via SSH primary`)
		return 2
	}
	switch args[0] {
	case "list-local", "local":
		runNodeCLI([]string{"release", "list-local"})
		return 0
	case "tags":
		runNodeCLI([]string{"git", "tags", "netductor"})
		return 0
	case "mirror-fetch", "mirror":
		runNodeCLI([]string{"git", "mirror-ensure", "netductor"})
		runNodeCLI([]string{"git", "mirror-fetch", "netductor"})
		return 0
	case "path":
		a := []string{"release", "path"}
		if len(args) >= 2 {
			a = append(a, args[1])
		}
		runNodeCLI(a)
		return 0
	case "prune":
		a := []string{"release", "prune"}
		a = append(a, args[1:]...)
		runNodeCLI(a)
		return 0
	case "build":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor-op release build <tag> [--skip-darwin]")
			return 2
		}
		a := []string{"release", "build", args[1]}
		a = append(a, args[2:]...)
		runNodeCLI(a)
		return 0
	case "build-host":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor-op release build-host <tag> [--skip-darwin]")
			return 2
		}
		skip := false
		for _, x := range args[2:] {
			if x == "--skip-darwin" {
				skip = true
			}
		}
		fmt.Fprintf(os.Stderr, "release build-host %s (local go → %s)\n", args[1], ndupdate.LocalReleasesDir())
		dst, err := ndupdate.BuildHost(ndupdate.BuildOpts{Tag: args[1], SkipDarwin: skip})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("ok", dst)
		fmt.Fprintln(os.Stderr, "hint: scp -r", dst, "root@primary:/var/lib/netductor/releases/  OR  netductor-op release … import via node")
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		return 2
	}
}
