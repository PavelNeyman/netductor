package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/sites"
)

func runSites(args []string) {
	if len(args) < 1 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		list, err := sites.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("no sites")
			return
		}
		for _, s := range list {
			fmt.Printf("%s\t%s\trpi=%s\tmt=%s\n", s.ID, s.Name, s.RPiID, s.MikroTikID)
		}
	case "rsc":
		id := ""
		if len(args) > 1 {
			id = args[1]
		}
		if id == "" {
			list, _ := sites.List()
			if len(list) > 0 {
				id = list[0].ID
			}
		}
		rsc, err := sites.RSCForSite(id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(rsc)
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor sites list|rsc [id]")
		os.Exit(2)
	}
}
