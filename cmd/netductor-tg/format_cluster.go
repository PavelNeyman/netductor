package main

import (
	"fmt"

	"github.com/PavelNeyman/netductor/internal/cluster"
)

// formatClusterStatusLine — one-liner for Status card.
func formatClusterStatusLine() string {
	v := cluster.Build()
	online, total := 0, len(v.Members)
	for _, m := range v.Members {
		if m.Online {
			online++
		}
	}
	role := "active"
	if !v.WriteAllowed {
		role = "standby"
	}
	if getLang() != "en" {
		return fmt.Sprintf("🧩 Кластер <b>%d</b>/%d · control <b>%s</b>", online, total, role)
	}
	return fmt.Sprintf("🧩 Cluster <b>%d</b>/%d · control <b>%s</b>", online, total, role)
}
