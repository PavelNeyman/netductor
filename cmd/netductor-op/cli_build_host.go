package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/operator"
)

func runDeployBuildHost(args []string) int {
	s := loadTUISettings()
	o := operator.BuildHostSpec{
		PrimaryHost: s.RemoteHost,
		PrimaryUser: orDefault(s.RemoteUser, "root"),
		PrimaryKey:  expandHome(orDefault(s.RemoteKey, "~/.ssh/netductor_primary")),
		BuildUser:   "root",
		Label:       "build-1",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host":
			if i+1 < len(args) {
				o.BuildHost = args[i+1]
				i++
			}
		case "--pass", "--password":
			if i+1 < len(args) {
				o.BuildPass = args[i+1]
				i++
			}
		case "--primary":
			if i+1 < len(args) {
				o.PrimaryHost = args[i+1]
				i++
			}
		case "--key":
			if i+1 < len(args) {
				o.PrimaryKey = expandHome(args[i+1])
				i++
			}
		case "--label":
			if i+1 < len(args) {
				o.Label = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("usage: netductor-op deploy build-host --host IP --pass PASS [--primary IP] [--key PATH]")
			return 0
		}
	}
	if o.BuildHost == "" || o.PrimaryHost == "" {
		fmt.Fprintln(os.Stderr, "primary (settings or --primary) and --host required")
		return 2
	}
	if err := operator.DeployBuildHost(o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
