package main

import (
	"encoding/json"
	"fmt"
	"github.com/PavelNeyman/netductor/internal/addons"
	"github.com/PavelNeyman/netductor/internal/format"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"os"
	"os/exec"
	"strings"
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
	st := addons.ListAddons()
	if vers, ok := st["versions"].(map[string]any); ok {
		if lp, ok := vers["lampac"].(map[string]any); ok {
			b.WriteString("\nlampac <code>")
			b.WriteString(fmt.Sprint(lp["running"]))
			b.WriteString("</code>")
			if lp["update_available"] == true {
				b.WriteString(" · update")
			}
		}
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
		polLabel := T("pol_btn")
		b.WriteString(fmt.Sprintf(`<tg-button-row align="left"><tg-button type="callback_data" data="e:policy:%s">%s %s</tg-button></tg-button-row>`+"\n", id, polLabel, esc(id)))
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
			b.WriteString("<i>Для Admin SPA / API (Authorization: Bearer …)\nСообщение исчезнет ~2 мин</i>")
		} else {
			b.WriteString("<i>For Admin SPA / API (Authorization: Bearer …)\nMessage auto-deletes ~2 min</i>")
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

	// firewall status (critical)
	b.WriteString(formatFirewallBlock() + nl)

	if ru {
		b.WriteString(fmt.Sprintf("🗂 Ноды: <b>%d</b> online / %d · 👥 VPN: <b>%d</b>"+nl, online, len(rows), vpnN))
	} else {
		b.WriteString(fmt.Sprintf("🗂 Nodes: <b>%d</b> online / %d · 👥 VPN: <b>%d</b>"+nl, online, len(rows), vpnN))
	}

	// Channel summary (detail screen: m:channel → tables)
	b.WriteString(nl + formatChannelSummaryHTML() + nl)
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func formatFirewallBlock() string {
	nl := string([]byte{10})
	ru := getLang() != "en"
	out, err := exec.Command("netductor", "firewall", "status", "--json").CombinedOutput()
	title := "🔥 <b>Firewall</b>"
	if ru {
		title = "🔥 <b>Файервол</b>"
	}
	if err != nil && len(out) == 0 {
		if ru {
			return title + nl + "🔴 <b>нет данных</b> (запустите netductor firewall apply)"
		}
		return title + nl + "🔴 <b>no data</b> (run netductor firewall apply)"
	}
	var st struct {
		Backend    string   `json:"backend"`
		Active     bool     `json:"active"`
		OK         bool     `json:"ok"`
		Role       string   `json:"role"`
		OpenWAN    []string `json:"open_wan"`
		DenyWAN    []string `json:"deny_wan"`
		Restricted []string `json:"restricted"`
		Warnings   []string `json:"warnings"`
		Detail     string   `json:"detail"`
	}
	_ = json.Unmarshal(out, &st)
	icon := "🟢"
	if !st.OK {
		icon = "🔴"
	} else if !st.Active {
		icon = "🟡"
	}
	var b strings.Builder
	b.WriteString(title + nl)
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString(fmt.Sprintf("<tr><td>status</td><td>%s ok=%v</td></tr>"+nl, icon, st.OK))
	b.WriteString(fmt.Sprintf("<tr><td>backend</td><td>%s</td></tr>"+nl, esc(st.Backend)))
	b.WriteString(fmt.Sprintf("<tr><td>active</td><td>%v</td></tr>"+nl, st.Active))
	b.WriteString(fmt.Sprintf("<tr><td>role</td><td>%s</td></tr>"+nl, esc(st.Role)))
	if len(st.OpenWAN) > 0 {
		b.WriteString(fmt.Sprintf("<tr><td>open WAN</td><td>%s</td></tr>"+nl, esc(strings.Join(st.OpenWAN, ", "))))
	}
	if len(st.DenyWAN) > 0 {
		b.WriteString(fmt.Sprintf("<tr><td>deny WAN</td><td>%s</td></tr>"+nl, esc(strings.Join(st.DenyWAN, ", "))))
	}
	if len(st.Restricted) > 0 {
		b.WriteString(fmt.Sprintf("<tr><td>restricted</td><td>%s</td></tr>"+nl, esc(strings.Join(st.Restricted, "; "))))
	}
	b.WriteString("</table>")
	for _, w := range st.Warnings {
		b.WriteString(nl + "⚠️ " + esc(w))
	}
	return b.String()
}


// formatChannelSummaryHTML — compact Status-card block (no raw dump).
func formatChannelSummaryHTML() string {
	ru := getLang() != "en"
	title := "📶 <b>Channel</b>"
	if ru {
		title = "📶 <b>Канал</b>"
	}
	rep, err := loadChannelReport()
	if err != nil {
		if ru {
			return title + "\n<i>нет данных</i>"
		}
		return title + "\n<i>no data</i>"
	}
	okN, total := 0, len(rep.Secondaries)
	for _, s := range rep.Secondaries {
		if s.Online && s.UplinkOK {
			okN++
		}
	}
	pathOK, pathT := 0, len(rep.Path)
	for _, p := range rep.Path {
		if p.PathOK {
			pathOK++
		}
	}
	nl := "\n"
	var b strings.Builder
	b.WriteString(title + nl)
	if ru {
		b.WriteString(fmt.Sprintf("• secondary uplink: <b>%d</b>/%d · path e2e: <b>%d</b>/%d"+nl, okN, total, pathOK, pathT))
		b.WriteString(fmt.Sprintf("• mismatch 30m: <b>%d</b> · reality 15m: <b>%d</b> (sec <b>%d</b>)"+nl, rep.MismatchLocal30m, rep.RealityInvalidTotal15m, rep.RealityInvalidFromSec15m))
	} else {
		b.WriteString(fmt.Sprintf("• secondary uplink: <b>%d</b>/%d · path e2e: <b>%d</b>/%d"+nl, okN, total, pathOK, pathT))
		b.WriteString(fmt.Sprintf("• mismatch 30m: <b>%d</b> · reality 15m: <b>%d</b> (from sec <b>%d</b>)"+nl, rep.MismatchLocal30m, rep.RealityInvalidTotal15m, rep.RealityInvalidFromSec15m))
	}
	return b.String()
}

type channelReportJSON struct {
	TS                       string `json:"ts"`
	MismatchLocal30m         int    `json:"mismatch_local_30m"`
	RealityInvalidFromSec15m int    `json:"reality_invalid_from_secondary_15m"`
	RealityInvalidTotal15m   int    `json:"reality_invalid_total_15m"`
	Secondaries              []struct {
		ID              string  `json:"id"`
		Name            string  `json:"name"`
		PublicIP        string  `json:"public_ip"`
		Online          bool    `json:"online"`
		HeartbeatAgeSec int     `json:"heartbeat_age_sec"`
		SingBoxOK       bool    `json:"singbox_ok"`
		UplinkOK        bool    `json:"uplink_ok"`
		TCP443OK        bool    `json:"tcp443_ok"`
		TCP443ms        float64 `json:"tcp443_ms"`
		Mismatch30m     int     `json:"mismatch_30m"`
	} `json:"secondaries"`
	Path []struct {
		SecondaryID string  `json:"secondary_id"`
		Name        string  `json:"name"`
		Online      bool    `json:"online"`
		Face443     bool    `json:"face_443"`
		Face443ms   float64 `json:"face_443_ms"`
		UplinkOK    bool    `json:"uplink_ok"`
		SingBoxOK   bool    `json:"singbox_ok"`
		SSH52222    bool    `json:"ssh_52222"`
		ICMP        bool    `json:"icmp_ok"`
		PathOK      bool    `json:"path_ok"`
		MgmtOK      bool    `json:"mgmt_ok"`
		Note        string  `json:"note"`
	} `json:"path_e2e"`
}

func loadChannelReport() (*channelReportJSON, error) {
	out, err := exec.Command(netductorBin(), "channel", "status", "--json").CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	var rep channelReportJSON
	if json.Unmarshal(out, &rep) != nil {
		return nil, fmt.Errorf("parse channel json")
	}
	return &rep, nil
}

func markOK(ok bool) string {
	if ok {
		return "✅"
	}
	return "🔴"
}

// formatChannelDetailHTML — full Channel screen: tables, no raw text dump.
func formatChannelDetailHTML() string {
	ru := getLang() != "en"
	nl := "\n"
	title := "📶 <b>Channel</b>"
	if ru {
		title = "📶 <b>Канал</b>"
	}
	rep, err := loadChannelReport()
	if err != nil {
		if ru {
			return title + nl + "<i>нет данных (channel status --json)</i>"
		}
		return title + nl + "<i>no data (channel status --json)</i>"
	}
	var b strings.Builder
	b.WriteString(title + nl)
	if ru {
		b.WriteString(fmt.Sprintf("<i>mismatch 30m local=<b>%d</b> · reality 15m total=<b>%d</b> / from secondary=<b>%d</b></i>"+nl+nl, rep.MismatchLocal30m, rep.RealityInvalidTotal15m, rep.RealityInvalidFromSec15m))
	} else {
		b.WriteString(fmt.Sprintf("<i>mismatch 30m local=<b>%d</b> · reality 15m total=<b>%d</b> / from secondary=<b>%d</b></i>"+nl+nl, rep.MismatchLocal30m, rep.RealityInvalidTotal15m, rep.RealityInvalidFromSec15m))
	}

	// Secondaries table
	h1 := "Secondary"
	if ru {
		h1 = "Secondary"
	}
	b.WriteString("<b>" + h1 + "</b>" + nl)
	b.WriteString(`<table bordered striped compact><tr><th>name</th><th>online</th><th>uplink</th><th>face:443</th><th>sb</th><th>hb</th></tr>`)
	for _, s := range rep.Secondaries {
		name := s.Name
		if name == "" {
			name = s.ID
		}
		if s.PublicIP != "" {
			name = name + "\n" + s.PublicIP
		}
		face := markOK(s.TCP443OK)
		if s.TCP443OK && s.TCP443ms > 0 {
			face = fmt.Sprintf("%s %.0fms", markOK(true), s.TCP443ms)
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%ds</td></tr>",
			esc(name), markOK(s.Online), markOK(s.UplinkOK), face, markOK(s.SingBoxOK), s.HeartbeatAgeSec))
	}
	if len(rep.Secondaries) == 0 {
		b.WriteString(`<tr><td colspan="6"><i>—</i></td></tr>`)
	}
	b.WriteString(`</table>` + nl + nl)

	// Path e2e table
	b.WriteString("<b>Path e2e</b>" + nl)
	b.WriteString(`<table bordered striped compact><tr><th>name</th><th>path</th><th>mgmt</th><th>face</th><th>ssh</th><th>note</th></tr>`)
	for _, p := range rep.Path {
		name := p.Name
		if name == "" {
			name = p.SecondaryID
		}
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			esc(name), markOK(p.PathOK), markOK(p.MgmtOK), markOK(p.Face443), markOK(p.SSH52222), esc(p.Note)))
	}
	if len(rep.Path) == 0 {
		b.WriteString(`<tr><td colspan="6"><i>—</i></td></tr>`)
	}
	b.WriteString(`</table>`)
	return b.String()
}
