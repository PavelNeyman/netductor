package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/ci"
)

func runCI(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor ci status
  netductor ci test <workdir>
  netductor ci exec [--image IMG] [--workdir DIR] -- <shell script>`)
		return 2
	}
	switch args[0] {
	case "status":
		fmt.Println(ci.StatusLine())
		if eng := ci.Engine(); eng == "" {
			fmt.Println("hint: install docker or podman for isolated builds")
		}
		if !ci.IsolationEnabled() {
			fmt.Println("WARN: NETDUCTOR_CI_HOST=1 — builds run on host")
		}
		return 0
	case "test":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor ci test <workdir>")
			return 2
		}
		out, err := ci.TestWorktree(args[1])
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "exec":
		img, wd, script := "", ".", ""
		rest := args[1:]
		for len(rest) > 0 {
			if rest[0] == "--image" && len(rest) >= 2 {
				img = rest[1]
				rest = rest[2:]
				continue
			}
			if rest[0] == "--workdir" && len(rest) >= 2 {
				wd = rest[1]
				rest = rest[2:]
				continue
			}
			if rest[0] == "--" {
				rest = rest[1:]
				script = joinArgs(rest)
				rest = nil
				break
			}
			script = joinArgs(rest)
			break
		}
		if script == "" {
			fmt.Fprintln(os.Stderr, "missing script")
			return 2
		}
		out, err := ci.Exec(ci.ExecOpts{Image: img, WorkDir: wd, Script: script})
		fmt.Print(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		return 2
	}
}

func joinArgs(a []string) string {
	if len(a) == 0 {
		return ""
	}
	out := a[0]
	for i := 1; i < len(a); i++ {
		out += " " + a[i]
	}
	return out
}
