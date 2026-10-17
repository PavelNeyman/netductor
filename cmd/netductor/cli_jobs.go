package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/jobs"
)

func runJobsCLI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor job list [--status running] | get <id> | cancel <id> | start <type> [k=v…]")
		os.Exit(2)
	}
	st := primaryJobStore()
	switch args[0] {
	case "list":
		status := ""
		typ := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--status" && i+1 < len(args) {
				i++
				status = args[i]
			}
			if args[i] == "--type" && i+1 < len(args) {
				i++
				typ = args[i]
			}
		}
		list := st.List(jobs.Status(status), typ)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"jobs": list})
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job get <id>")
			os.Exit(2)
		}
		j := st.Get(args[1])
		if j == nil {
			fmt.Fprintln(os.Stderr, "not found")
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(j.Snapshot())
	case "cancel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job cancel <id>")
			os.Exit(2)
		}
		if err := st.Cancel(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok")
	case "start":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job start <type> [key=val…]")
			os.Exit(2)
		}
		typ := args[1]
		params := map[string]any{}
		for _, a := range args[2:] {
			k, v, ok := strings.Cut(a, "=")
			if ok {
				params[k] = v
			}
		}
		j, err := st.Start(typ, params)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(j.ID)
	default:
		fmt.Fprintln(os.Stderr, "unknown")
		os.Exit(2)
	}
}
