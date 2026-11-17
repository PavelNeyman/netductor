package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/cluster"
)

func formatClusterHTML() string {
	ru := getLang() != "en"
	v := cluster.Build()
	nl := string([]byte{10})
	var b strings.Builder
	if ru {
		b.WriteString("🧩 <b>Кластер</b>" + nl)
	} else {
		b.WriteString("🧩 <b>Cluster</b>" + nl)
	}
	leader := v.LeaderID
	if leader == "" {
		leader = "—"
	}
	wa := "no"
	if v.WriteAllowed {
		wa = "yes"
	}
	if ru {
		b.WriteString(fmt.Sprintf("leader <code>%s</code> · write <b>%s</b>"+nl+nl, esc(leader), wa))
	} else {
		b.WriteString(fmt.Sprintf("leader <code>%s</code> · write <b>%s</b>"+nl+nl, esc(leader), wa))
	}
	b.WriteString("<table bordered striped compact>" + nl)
	if ru {
		b.WriteString("<tr><th>тип</th><th>id</th><th>online</th><th>детали</th></tr>" + nl)
	} else {
		b.WriteString("<tr><th>kind</th><th>id</th><th>online</th><th>detail</th></tr>" + nl)
	}
	for _, m := range v.Members {
		on := "🔴"
		if m.Online {
			on = "🟢"
		}
		id := m.ID
		if len(id) > 24 {
			id = id[:10] + "…" + id[len(id)-8:]
		}
		det := m.PublicIP
		if m.Version != "" {
			if det != "" {
				det += " · "
			}
			det += "v" + m.Version
		}
		if m.Kind == "control" {
			det = m.ControlRole
			if m.IsLeader {
				det += " · leader"
			}
		}
		if m.Kind == "egress" {
			sb, up := "sb✗", "up✗"
			if m.SingBoxOK {
				sb = "sb✓"
			}
			if m.UplinkOK {
				up = "up✓"
			}
			if det != "" {
				det += " · "
			}
			det += sb + " " + up
		}
		if det == "" {
			det = "—"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td><td>%s</td><td>%s</td></tr>"+nl,
			esc(m.Kind), esc(id), on, esc(det)))
	}
	b.WriteString("</table>")
	return b.String()
}

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
