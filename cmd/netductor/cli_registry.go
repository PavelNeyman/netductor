package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/registry"
)

func runRegistry(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor registry status
  netductor registry ensure
  netductor registry stop
  netductor registry crane
  netductor registry catalog
  netductor registry auth-set <user> <pass>
  netductor registry auth-clear`)
		return 2
	}
	switch args[0] {
	case "status":
		st := registry.StatusInfo()
		fmt.Printf("ok=%v running=%v http=%v addr=%s engine=%s crane=%s\n",
			st.OK, st.Running, st.HTTPReach, st.Addr, st.Engine, st.CranePath)
		if st.Error != "" {
			fmt.Println("error:", st.Error)
		}
		if st.Hint != "" {
			fmt.Println("hint:", st.Hint)
		}
		return 0
	case "ensure":
		st, err := registry.Ensure()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("registry ok addr=%s data=%s\n", st.Addr, st.DataDir)
		return 0
	case "stop":
		if err := registry.Stop(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("stopped")
		return 0
	case "crane":
		p, err := registry.EnsureCrane()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(p)
		return 0
	case "auth-set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor registry auth-set <user> <pass>")
			return 2
		}
		if err := registry.SetAuth(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("auth set; registry restarted")
		return 0
	case "auth-clear":
		if err := registry.ClearAuth(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("auth cleared")
		return 0
	case "catalog":
		list, err := registry.CatalogDetail()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, r := range list {
			if len(r.Tags) == 0 {
				fmt.Println(r.Name)
			} else {
				fmt.Printf("%s\t%s\n", r.Name, strings.Join(r.Tags, ", "))
			}
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		return 2
	}
}
