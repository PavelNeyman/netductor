package main

import (
	"fmt"
	"os"
	"strconv"

	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func runGit(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor git list
  netductor git init <name>
  netductor git log <name> [n]
  netductor git show <name> [rev]
  netductor git root`)
		return 2
	}
	switch args[0] {
	case "root":
		fmt.Println(gitstore.Root())
		return 0
	case "list":
		list, err := gitstore.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, n := range list {
			fmt.Println(n)
		}
		return 0
	case "init":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git init <name>")
			return 2
		}
		path, err := gitstore.Init(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(path)
		fmt.Fprintf(os.Stderr, "remote: ssh://root@HOST:%d%s\n", 52222, path)
		return 0
	case "log":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git log <name> [n]")
			return 2
		}
		n := 20
		if len(args) >= 3 {
			n, _ = strconv.Atoi(args[2])
		}
		out, err := gitstore.Log(args[1], n)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "show":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git show <name> [rev]")
			return 2
		}
		rev := "HEAD"
		if len(args) >= 3 {
			rev = args[2]
		}
		out, err := gitstore.Show(args[1], rev)
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown git subcommand")
		return 2
	}
}
