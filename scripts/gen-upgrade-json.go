//go:build ignore

// Generate upgrade.json from in-code floors (CF-15). Usage:
//
//	go run scripts/gen-upgrade-json.go [version]
package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/upgrade"
	ndver "github.com/PavelNeyman/netductor/internal/version"
)

func main() {
	ver := ndver.Release
	if len(os.Args) > 1 {
		ver = os.Args[1]
	}
	b, err := upgrade.MarshalMetaJSON(ver)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))
}
