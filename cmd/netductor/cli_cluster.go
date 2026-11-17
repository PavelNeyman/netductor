package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/cluster"
)

func runCluster(args []string) {
	if len(args) < 1 || args[0] == "status" || args[0] == "view" {
		v := cluster.Build()
		fmt.Printf("leader=%s write_allowed=%v ts=%s\n", v.LeaderID, v.WriteAllowed, v.TS)
		for _, m := range v.Members {
			line := fmt.Sprintf("  %-8s %-28s online=%v", m.Kind, m.ID, m.Online)
			if m.PublicIP != "" {
				line += " ip=" + m.PublicIP
			}
			if m.Version != "" {
				line += " ver=" + m.Version
			}
			if m.Kind == "control" {
				line += fmt.Sprintf(" role=%s leader=%v", m.ControlRole, m.IsLeader)
			}
			if m.Kind == "egress" {
				line += fmt.Sprintf(" sb=%v uplink=%v", m.SingBoxOK, m.UplinkOK)
			}
			fmt.Println(line)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "usage: netductor cluster status")
	os.Exit(2)
}
