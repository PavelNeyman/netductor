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
  netductor release path [tag]           # print store root or tag dir`)
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
	default:
		fmt.Fprintln(os.Stderr, "unknown release subcommand:", args[0])
		return 2
	}
}
