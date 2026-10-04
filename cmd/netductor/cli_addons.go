package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/addons"
)

func runAddons(args []string) {
	if len(args) == 0 || args[0] == "list" {
		b, _ := json.MarshalIndent(addons.ListAddons(), "", "  ")
		fmt.Println(string(b))
		return
	}
	switch args[0] {
	case "lampac":
		b, _ := json.MarshalIndent(addons.CollectLampac(), "", "  ")
		fmt.Println(string(b))
	default:
		if len(args) > 0 && (args[0] == "update" || args[0] == "upgrade") {
		name := "all"
		if len(args) > 1 {
			name = args[1]
		}
		out, err := addons.Update(name)
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "usage: netductor addons [list|lampac|update [name|all]]")
		os.Exit(2)
	}
}
