package main

import (
	"fmt"
	"os"
)

func runOpRelease(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage (SSH to primary → netductor …):
  netductor-op release list-local
  netductor-op release tags
  netductor-op release mirror-fetch
  netductor-op release build <tag> [--skip-darwin]
  netductor-op release path [tag]`)
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
	case "build":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor-op release build <tag> [--skip-darwin]")
			return 2
		}
		a := []string{"release", "build", args[1]}
		for _, x := range args[2:] {
			a = append(a, x)
		}
		runNodeCLI(a)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		return 2
	}
}
