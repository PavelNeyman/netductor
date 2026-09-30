package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/stack"
)

func formatFleetDigestHTML() string {
	d := fleet.Build()
	ru := getLang() != "en"
	nl := "\n"
	var b strings.Builder
	if ru {
		b.WriteString("📡 <b>Fleet health</b>" + nl)
	} else {
		b.WriteString("📡 <b>Fleet health</b>" + nl)
	}
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString(fmt.Sprintf("<tr><td>primary</td><td><code>%s</code></td></tr>"+nl, esc(d.PrimaryVersion)))
	if d.UpdateAvailable {
		b.WriteString(fmt.Sprintf("<tr><td>update</td><td>🆕 %s</td></tr>"+nl, esc(d.LatestRelease)))
	} else {
		b.WriteString("<tr><td>update</td><td>✅ up to date</td></tr>" + nl)
	}
	for u, st := range d.Services {
		mark := "🔴"
		if st == "active" {
			mark = "🟢"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s %s</td></tr>"+nl, esc(u), mark, esc(st)))
	}
	if d.BackupAgeHours != nil {
		b.WriteString(fmt.Sprintf("<tr><td>backup age</td><td>%.1fh</td></tr>"+nl, *d.BackupAgeHours))
	}
	if len(d.LEHosts) > 0 {
		b.WriteString(fmt.Sprintf("<tr><td>LE</td><td><code>%s</code></td></tr>"+nl, esc(strings.Join(d.LEHosts, ", "))))
	}
	b.WriteString(fmt.Sprintf("<tr><td>vpn users</td><td>%d</td></tr>"+nl, d.VPNUsers))
	b.WriteString("</table>" + nl)
	if len(d.Secondaries) > 0 {
		b.WriteString(nl + "<b>Secondary</b>" + nl)
		b.WriteString("<table bordered striped compact>" + nl)
		b.WriteString("<tr><th>name</th><th>ver</th><th>status</th></tr>" + nl)
		for _, s := range d.Secondaries {
			label := s.Name
			if label == "" {
				label = s.ID
				if len(label) > 12 {
					label = label[:8] + "…"
				}
			}
			st := "🔴 off"
			if s.Online {
				st = "🟢"
				if !s.SingBox {
					st += " sb?"
				}
				if !s.Uplink {
					st += " up?"
				}
			}
			ver := s.Version
			if ver == "" {
				ver = "—"
			}
			b.WriteString(fmt.Sprintf("<tr><td>%s</td><td><code>%s</code></td><td>%s</td></tr>"+nl, esc(label), esc(ver), esc(st)))
		}
		b.WriteString("</table>" + nl)
	}
	if len(d.VPNNearQuota) > 0 {
		if ru {
			b.WriteString(nl + "<i>Soft limit настроен: " + esc(strings.Join(d.VPNNearQuota, ", ")) + "</i>" + nl)
		} else {
			b.WriteString(nl + "<i>Soft limit set: " + esc(strings.Join(d.VPNNearQuota, ", ")) + "</i>" + nl)
		}
	}
		// stack units
	st := stack.Collect()
	b.WriteString(nl + stack.FormatHTML(st) + nl)
	var bad []string
	for _, u := range st.Units {
		if !u.OK && u.Unit != "netductor-redirect" && u.Unit != "netductor-stack-watchdog.timer" && u.Unit != "netductor-backup.timer" {
			// optional timers may be inactive — only flag core
			if u.Unit == "netductor-api" || u.Unit == "netductor-telegram-bot" || u.Unit == "sing-box" || u.Unit == "blocky" {
				bad = append(bad, u.Unit+":"+u.Active)
			}
		}
	}
	if len(bad) > 0 {
		if ru {
			b.WriteString("⚠️ <b>Core units</b>: <code>" + esc(strings.Join(bad, ", ")) + "</code>" + nl)
		} else {
			b.WriteString("⚠️ <b>Core units</b>: <code>" + esc(strings.Join(bad, ", ")) + "</code>" + nl)
		}
		b.WriteString(`<tg-button-row><tg-button type="callback_data" style="primary" data="m:stack">Stack</tg-button><tg-button type="callback_data" data="m:stack:watchdog">WD</tg-button></tg-button-row>` + nl)
	} else {
		b.WriteString(`<tg-button-row><tg-button type="callback_data" data="m:stack">🧱 Stack</tg-button></tg-button-row>` + nl)
	}

// disaster path short
	if ru {
		b.WriteString(nl + "<i>DR: netductor recover --from-secondary … · Updates · doctor</i>")
	} else {
		b.WriteString(nl + "<i>DR: netductor recover --from-secondary … · Updates · doctor</i>")
	}
	b.WriteString(nl + `<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="m:digest">🔄</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:updates">Updates</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:dr">DR</tg-button>`)
	b.WriteString(`</tg-button-row>`)
	return b.String()
}

func formatDisasterHTML() string {
	ru := getLang() != "en"
	nl := "\n"
	var b strings.Builder
	if ru {
		b.WriteString("🛟 <b>Disaster recovery</b>" + nl)
		b.WriteString("<i>Primary мёртв — данные на secondary (backup_pull).</i>" + nl + nl)
		b.WriteString("<b>Чеклист</b>" + nl)
		b.WriteString("☐ 1. Доступ к secondary (SSH 52222 / console)" + nl)
		b.WriteString("☐ 2. <code>netductor recover --from-secondary URL --recovery-token T --key KEY</code>" + nl)
		b.WriteString("☐ 3. doctor · LE · vpn apply · SP/PS" + nl)
		b.WriteString("☐ 4. secondary agent heartbeat" + nl)
		b.WriteString("☐ 5. Update apply при необходимости" + nl)
	} else {
		b.WriteString("🛟 <b>Disaster recovery</b>" + nl)
		b.WriteString("<i>Primary down — encrypted peers on secondary (backup_pull).</i>" + nl + nl)
		b.WriteString("<b>Checklist</b>" + nl)
		b.WriteString("☐ 1. Reach secondary (SSH 52222 / console)" + nl)
		b.WriteString("☐ 2. <code>netductor recover --from-secondary URL --recovery-token T --key KEY</code>" + nl)
		b.WriteString("☐ 3. doctor · LE · vpn apply · SP/PS" + nl)
		b.WriteString("☐ 4. secondary agent heartbeat" + nl)
		b.WriteString("☐ 5. Update apply if needed" + nl)
	}
	b.WriteString(nl + `<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="m:digest">Fleet</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" data="m:menu">Menu</tg-button>`)
	b.WriteString(`</tg-button-row>`)
	return b.String()
}
