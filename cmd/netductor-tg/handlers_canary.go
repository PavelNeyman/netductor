package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleCanaryCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:canary" && !strings.HasPrefix(data, "m:canary:") && !strings.HasPrefix(data, "u:canary:") {
		return false
	}
	if strings.HasPrefix(data, "u:canary:") {
		name := strings.TrimPrefix(data, "u:canary:")
		on, err := vpn.ToggleCanary(name)
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), userHubKeyboard(name))
			return true
		}
		mark := "removed from canary"
		if on {
			mark = "added to canary"
		}
		if getLang() != "en" {
			if on {
				mark = "добавлен в canary"
			} else {
				mark = "убран из canary"
			}
		}
		reply(token, chat, msgID, "✅ <b>"+esc(name)+"</b> — "+mark+"\n\n"+formatUserHubHTML(name), userHubKeyboard(name))
		return true
	}
	if data == "m:canary" || data == "m:canary:refresh" {
		reply(token, chat, msgID, formatCanaryHub(), canaryHubKB())
		return true
	}
	if strings.HasPrefix(data, "m:canary:toggle:") {
		name := strings.TrimPrefix(data, "m:canary:toggle:")
		_, _ = vpn.ToggleCanary(name)
		reply(token, chat, msgID, formatCanaryHub(), canaryHubKB())
		return true
	}
	return true
}

func formatCanaryHub() string {
	list := vpn.LoadCanaryUsers()
	st10 := vpn.CollectMismatch(10)
	st30 := vpn.CollectMismatch(30)
	var b strings.Builder
	if getLang() != "en" {
		b.WriteString("🐦 <b>Canary users</b>\n")
		b.WriteString("<i>«Свои» VPN-пользователи для алертов flow mismatch. ")
		b.WriteString("Сканеры и чужие IP не должны поднимать тревогу — только сбои Vision у этих имён.</i>\n\n")
		b.WriteString("Алерты смотрят <b>canary</b>-счётчик, если список не пуст.\n")
	} else {
		b.WriteString("🐦 <b>Canary users</b>\n")
		b.WriteString("<i>«Our» VPN users for flow-mismatch alerts. ")
		b.WriteString("Scanners must not page you — only Vision failures for these names.</i>\n\n")
		b.WriteString("When the list is non-empty, alerts use the <b>canary</b> counter.\n")
	}
	b.WriteString(fmt.Sprintf("\n📊 mismatch 10m: total=<b>%d</b> canary=<b>%d</b> other=<b>%d</b>\n",
		st10.Total, st10.CanaryTotal, st10.OtherTotal))
	b.WriteString(fmt.Sprintf("📊 mismatch 30m: total=<b>%d</b> canary=<b>%d</b> other=<b>%d</b>\n\n",
		st30.Total, st30.CanaryTotal, st30.OtherTotal))
	if len(list) == 0 {
		b.WriteString("<i>List empty — spikes use global thresholds.</i>\n")
		b.WriteString("Add from a user card or toggle below.\n")
	} else {
		b.WriteString("<b>Members:</b>\n")
		for _, u := range list {
			b.WriteString("• <code>" + esc(u) + "</code>\n")
		}
	}
	b.WriteString("\n<code>/etc/netductor/canary-users.json</code>")
	return b.String()
}

func canaryHubKB() map[string]any {
	rows := [][]map[string]any{
		{btn("🔄", "m:canary:refresh", ""), btn("« Users", "m:users", ""), btn(T("main_menu"), "m:menu", "")},
	}
	users, _ := vpn.List()
	in := map[string]bool{}
	for _, u := range vpn.LoadCanaryUsers() {
		in[u] = true
	}
	for _, u := range users {
		if u.Name == "relay-uplink" || vpn.IsEdgeUser(u.Name) {
			continue
		}
		mark := "☐ " + u.Name
		if in[u.Name] {
			mark = "☑ " + u.Name
		}
		rows = append(rows, []map[string]any{btn(mark, "m:canary:toggle:"+u.Name, "")})
	}
	return map[string]any{"inline_keyboard": rows}
}
