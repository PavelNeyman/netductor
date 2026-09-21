package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/registry"
)

func runRegistry(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `usage:
  netductor registry status
  netductor registry ensure
  netductor registry stop
  netductor registry crane
  netductor registry catalog`)
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
	case "catalog":
		list, err := registry.Catalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, n := range list {
			fmt.Println(n)
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		return 2
	}
}
