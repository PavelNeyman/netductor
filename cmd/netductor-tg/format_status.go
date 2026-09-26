package main

import (
	"github.com/PavelNeyman/netductor/internal/format"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/PavelNeyman/netductor/internal/addons"
)

func yn(ok bool, good, bad string) string {
	if ok {
		return good
	}
	return bad
}

func formatLampacHTML() string {
	ru := getLang() != "en"
	st := addons.CollectLampac()
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("📺 <b>Lampac</b>" + nl)
	b.WriteString("<table bordered striped compact>" + nl)
	if ru {
		b.WriteString("<tr><th>параметр</th><th>значение</th></tr>" + nl)
		b.WriteString(fmt.Sprintf("<tr><td>контейнер</td><td>%s %s</td></tr>"+nl, yn(st.Running, "🟢", "🔴"), yn(st.Running, "запущен", "остановлен")))
		b.WriteString(fmt.Sprintf("<tr><td>health</td><td>%s %s</td></tr>"+nl, yn(st.Healthy, "✅", "⚠️"), yn(st.Healthy, "ok", "bad")))
	} else {
		b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
		b.WriteString(fmt.Sprintf("<tr><td>container</td><td>%s %s</td></tr>"+nl, yn(st.Running, "🟢", "🔴"), yn(st.Running, "running", "stopped")))
		b.WriteString(fmt.Sprintf("<tr><td>health</td><td>%s %s</td></tr>"+nl, yn(st.Healthy, "✅", "⚠️"), yn(st.Healthy, "ok", "bad")))
	}
	if st.Image != "" {
		b.WriteString(fmt.Sprintf("<tr><td>image</td><td><code>%s</code></td></tr>"+nl, esc(st.Image)))
	}
	b.WriteString(fmt.Sprintf("<tr><td>bind</td><td><code>%s</code></td></tr>"+nl, esc(st.Bind)))
	if st.VersionHash != "" {
		b.WriteString(fmt.Sprintf("<tr><td>version</td><td><code>%s</code></td></tr>"+nl, esc(st.VersionHash)))
	}
	if st.CPU != "" || st.Mem != "" {
		b.WriteString(fmt.Sprintf("<tr><td>cpu/ram</td><td>%s · %s</td></tr>"+nl, esc(st.CPU), esc(st.Mem)))
	}
	b.WriteString(fmt.Sprintf("<tr><td>ping</td><td>%s</td></tr>"+nl, yn(st.PingOK, "✅", "—")))
	b.WriteString(fmt.Sprintf("<tr><td>chromium</td><td>%s</td></tr>"+nl, yn(st.ChromiumOK, "✅", "—")))
	b.WriteString(fmt.Sprintf("<tr><td>UI</td><td><code>%s</code></td></tr>"+nl, esc(st.UIURL)))
	b.WriteString(fmt.Sprintf("<tr><td>admin</td><td><code>%s</code></td></tr>"+nl, esc(st.AdminURL)))
	b.WriteString("</table>" + nl)
	if ru {
		b.WriteString("<i>Только localhost — доступ через VPN / SSH-туннель</i>")
	} else {
		b.WriteString("<i>Loopback only — reach via VPN / SSH tunnel</i>")
	}
	return b.String()
}


func formatAddonsHTML() string {
	ru := getLang() != "en"
	var b strings.Builder
	if ru {
		b.WriteString("🧩 <b>Аддоны</b>\n<i>Опциональные сервисы на primary</i>\n")
	} else {
		b.WriteString("🧩 <b>Addons</b>\n<i>Optional services on primary</i>\n")
	}
	b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" style="primary" data="m:addon:lampac">📺 Lampac</tg-button></tg-button-row>`)
	return b.String()
}

// formatEdgeListHTML turns tab-separated edge list into cards.
func formatEdgeListHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if getLang() != "en" {
			return "📡 <i>Нет устройств</i>"
		}
		return "📡 <i>No devices</i>"
	}
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("📡 <b>Роутеры / edge</b>\n\n")
	} else {
		b.WriteString("📡 <b>Routers / edge</b>\n\n")
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		// id status online board host ip=… last=
		id := parts[0]
		rest := ""
		if len(parts) > 1 {
			rest = strings.Join(parts[1:], " · ")
		}
		icon := "⚪"
		low := strings.ToLower(line)
		if strings.Contains(low, "online") {
			icon = "🟢"
		} else if strings.Contains(low, "offline") {
			icon = "🔴"
		} else if strings.Contains(low, "pending") {
			icon = "⏳"
		}
		b.WriteString(fmt.Sprintf("%s <code>%s</code>\n   %s\n", icon, esc(id), esc(rest)))
	}
	return b.String()
}

func formatPendingHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	nl := string([]byte{10})
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("⏳ <b>Ожидают approve</b>" + nl)
	} else {
		b.WriteString("⏳ <b>Pending enroll</b>" + nl)
	}
	if raw == "" {
		if getLang() != "en" {
		b.WriteString("<i>Список пуст</i>")
	} else {
		b.WriteString("<i>Empty</i>")
	}
		return b.String()
	}
	var ids []string
	var extras []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		ids = append(ids, parts[0])
		if len(parts) > 1 {
			extras = append(extras, strings.Join(parts[1:], " · "))
		} else {
			extras = append(extras, "")
		}
	}
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>#</th><th>id</th><th>info</th></tr>" + nl)
	for i, id := range ids {
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%s</td></tr>"+nl, i+1, esc(id), esc(extras[i])))
	}
	b.WriteString("</table>" + nl)
	// Compact: ✅n 🚫n pairs in rows
	const per = 3
	for i, id := range ids {
		if i%per == 0 {
			if i > 0 {
				b.WriteString(`</tg-button-row>` + nl)
			}
			b.WriteString(`<tg-button-row align="left">`)
		}
		n := i + 1
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="success" data="e:appr:%s">✅%d</tg-button>`, id, n))
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="danger" data="e:deny:%s">🚫%d</tg-button>`, id, n))
	}
	if len(ids) > 0 {
		b.WriteString(`</tg-button-row>` + nl)
	}
	return b.String()
}

func formatTemplatesHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("📋 <b>Шаблоны</b>\n\n")
	} else {
		b.WriteString("📋 <b>Templates</b>\n\n")
	}
	if raw == "" {
		b.WriteString("<i>—</i>")
		return b.String()
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b.WriteString("• <code>" + esc(line) + "</code>\n")
	}
	return b.String()
}

func formatSessionHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	// expect token line(s) from vpn session
	var token string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if len(line) >= 32 && isHexish(line) {
			token = line
			break
		}
	}
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("🔑 <b>Session token</b>\n\n")
	} else {
		b.WriteString("🔑 <b>Session token</b>\n\n")
	}
	if token != "" {
		b.WriteString("<code>" + esc(token) + "</code>\n\n")
		if getLang() != "en" {
			b.WriteString("<i>Для Admin SPA / API (Authorization: Bearer …)</i>")
		} else {
			b.WriteString("<i>For Admin SPA / API (Authorization: Bearer …)</i>")
		}
	} else {
		b.WriteString("<pre>" + esc(raw) + "</pre>")
	}
	return b.String()
}

func isHexish(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}




func formatStatusPretty() string {
	nl := string([]byte{10})
	ru := getLang() != "en"
	host, _ := os.Hostname()
	title := "📊 <b>Primary status</b>"
	if ru {
		title = "📊 <b>Статус primary</b>"
	}
	var b strings.Builder
	b.WriteString(title + nl)
	b.WriteString("<code>" + esc(host) + "</code>" + nl + nl)

	// metrics once (not repeated under Tools)
	raw, _ := json.Marshal(metrics.Collect())
	r := format.API("metrics", raw, catalogLang())
	b.WriteString(r.HTML + nl + nl)

	// services table
	svcTitle := "⚙️ <b>Services</b>"
	if ru {
		svcTitle = "⚙️ <b>Сервисы</b>"
	}
	b.WriteString(svcTitle + nl)
	b.WriteString("<table bordered striped compact>" + nl + "<tr><th>unit</th><th>st</th></tr>" + nl)
	for _, u := range []string{"sing-box", "netductor-api", "netductor-telegram-bot"} {
		out, _ := exec.Command("systemctl", "is-active", u).CombinedOutput()
		st := strings.TrimSpace(string(out))
		icon := "🔴"
		if st == "active" {
			icon = "🟢"
		}
		b.WriteString("<tr><td>" + esc(u) + "</td><td>" + icon + " " + esc(st) + "</td></tr>" + nl)
	}
	b.WriteString("</table>" + nl + nl)

	// short fleet summary — no full lists (Fleet / Users are separate)
	rows := parseNodesList()
	online := 0
	for _, r := range rows {
		if strings.EqualFold(r.Status, "online") {
			online++
		}
	}
	vpnN := 0
	if out := strings.TrimSpace(runVPN("list")); out != "" {
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "(") {
				vpnN++
			}
		}
	}
	if ru {
		b.WriteString(fmt.Sprintf("🗂 Ноды: <b>%d</b> online / %d · 👥 VPN: <b>%d</b>"+nl, online, len(rows), vpnN))
	} else {
		b.WriteString(fmt.Sprintf("🗂 Nodes: <b>%d</b> online / %d · 👥 VPN: <b>%d</b>"+nl, online, len(rows), vpnN))
	}
	b.WriteString(`<tg-button-row align="left">`)
	if ru {
		b.WriteString(`<tg-button type="callback_data" style="primary" data="m:fleet">🌐 Флот</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" data="m:users">👥 VPN</tg-button>`)
	} else {
		b.WriteString(`<tg-button type="callback_data" style="primary" data="m:fleet">🌐 Fleet</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" data="m:users">👥 VPN</tg-button>`)
	}
	b.WriteString(`</tg-button-row>`)
	return b.String()
}


func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

