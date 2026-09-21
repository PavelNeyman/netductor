package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/PavelNeyman/netductor/internal/hardening"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
)

func runGit(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor git list
  netductor git init <name>
  netductor git log <name> [n]
  netductor git show <name> [rev]
  netductor git pipelines
  netductor git pipeline <repo> <name> [args...]
  netductor git delete <name>
  netductor git root`)
		return 2
	}
	switch args[0] {
	case "root":
		fmt.Println(gitstore.Root())
		return 0
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor git delete <name>")
			return 2
		}
		if err := gitstore.Delete(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("deleted", args[1])
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
		fmt.Fprintln(os.Stderr, "remote:", gitstore.RemoteHint(args[1], hardening.SSHPort()))
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
	case "pipelines":
		_ = gitstore.EnsureSamplePipeline()
		list, err := gitstore.ListPipelines()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, n := range list {
			fmt.Println(n)
		}
		return 0
	case "pipeline":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor git pipeline <repo> <pipeline> [args...]")
			return 2
		}
		out, err := gitstore.RunPipeline(args[1], args[2], args[3:]...)
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
