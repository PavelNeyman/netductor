package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/dnsblock"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/sites"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleGuestCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	if data == "m:guest" {
		title := "⏱ <b>Гостевой доступ</b>\nВыберите срок для пользователя <code>guest</code>:"
		if !ru {
			title = "⏱ <b>Guest access</b>\nChoose TTL for shared <code>guest</code> user:"
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("10 мин", "m:guest:10m", ""), btn("1 час", "m:guest:1h", "")},
			{btn("24 часа", "m:guest:24h", ""), btn("7 дней", "m:guest:7d", "")},
			{btn(T("main_menu"), "m:menu", "primary")},
		}}
		if !ru {
			kb = map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("10 min", "m:guest:10m", ""), btn("1 hour", "m:guest:1h", "")},
				{btn("24 hours", "m:guest:24h", ""), btn("7 days", "m:guest:7d", "")},
				{btn(T("main_menu"), "m:menu", "primary")},
			}}
		}
		reply(token, chat, msgID, title, kb)
		return
	}
	ttl := time.Hour
	switch strings.TrimPrefix(data, "m:guest:") {
	case "10m":
		ttl = 10 * time.Minute
	case "1h":
		ttl = time.Hour
	case "24h":
		ttl = 24 * time.Hour
	case "7d":
		ttl = 7 * 24 * time.Hour
	}
	g, link, err := vpn.IssueGuestAccess(ttl, "tg")
	if err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), guestBackKB())
		return
	}
	// Prefer link from registry
	if link == "" {
		if users, e := vpn.List(); e == nil {
			for _, u := range users {
				if u.Name == "guest" && u.UUID != "" {
					link = vpn.PreferredVLESSLink("guest", u.UUID)
					break
				}
			}
		}
	}
	until := time.Unix(g.Expires, 0).In(time.FixedZone("MSK", 3*3600)).Format("2006-01-02 15:04 MSK")
	body := fmt.Sprintf("⏱ <b>Guest</b> до <code>%s</code>\n", until)
	if !ru {
		body = fmt.Sprintf("⏱ <b>Guest</b> until <code>%s</code>\n", until)
	}
	if link != "" {
		body += "<pre><code>" + esc(link) + "</code></pre>\n"
		body += "Users → guest → Access — QR."
	} else {
		body += "Users → <code>guest</code> → Access for QR/links."
	}
	reply(token, chat, msgID, body, guestBackKB())
}

func guestBackKB() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⏱ Guest", "m:guest", ""), btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func handleDNSCB(token string, chat int64, msgID int, data string) {
	if data == "m:dns" {
		showDNSMenu(token, chat, msgID, "")
		return
	}
	parts := strings.Split(data, ":")
	if len(parts) >= 4 && (parts[2] == "on" || parts[2] == "off") {
		on := parts[2] == "on"
		id := strings.Join(parts[3:], ":")
		if err := dnsblock.SetEnabled(id, on); err != nil {
			showDNSMenu(token, chat, msgID, "❌ "+err.Error())
			return
		}
		msg := "✅ " + id
		if on {
			msg += " ON"
		} else {
			msg += " OFF"
		}
		showDNSMenu(token, chat, msgID, msg)
		return
	}
	showDNSMenu(token, chat, msgID, "")
}

func showDNSMenu(token string, chat int64, msgID int, status string) {
	body := dnsblock.FormatCatalogHTML()
	if status != "" {
		body = status + "\n\n" + body
	}
	rows := [][]map[string]any{}
	for _, e := range dnsblock.Catalog() {
		mark := "☐"
		if e.Enabled {
			mark = "☑"
		}
		act := "m:dns:off:" + e.ID
		if !e.Enabled {
			act = "m:dns:on:" + e.ID
		}
		label := mark + " " + e.ID
		if len(label) > 40 {
			label = label[:40]
		}
		rows = append(rows, []map[string]any{btn(label, act, "")})
	}
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	reply(token, chat, msgID, body, map[string]any{"inline_keyboard": rows})
}

func handleBackupCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	s := install.LoadBackupSchedule()
	if data == "m:backup" {
		showBackupMenu(token, chat, msgID, s, "")
		return
	}
	if data == "m:backup:run" {
		_ = exec.Command("systemctl", "start", "netductor-backup.service").Start()
		msg := "✅ Бэкап запущен (systemd)"
		if !ru {
			msg = "✅ Backup started (systemd)"
		}
		showBackupMenu(token, chat, msgID, s, msg)
		return
	}
	if data == "m:backup:custom" {
		setState(chat, "wait_backup_time", "")
		hint := "Введите время <code>HH:MM</code> (UTC), например <code>01:30</code>"
		if !ru {
			hint = "Enter time <code>HH:MM</code> (UTC), e.g. <code>01:30</code>"
		}
		reply(token, chat, msgID, hint, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🗓 Backup", "m:backup", ""), btn(T("main_menu"), "m:menu", "primary")},
		}})
		return
	}
	if strings.HasPrefix(data, "m:backup:set:") {
		parts := strings.Split(data, ":")
		if len(parts) >= 5 {
			h, _ := strconv.Atoi(parts[3])
			m, _ := strconv.Atoi(parts[4])
			s = install.BackupSchedule{Hour: h, Minute: m, UTC: true}
			if err := install.SaveBackupSchedule(s); err != nil {
				showBackupMenu(token, chat, msgID, install.LoadBackupSchedule(), "❌ "+err.Error())
				return
			}
			msg := fmt.Sprintf("✅ %s", install.FormatBackupSchedule(s))
			showBackupMenu(token, chat, msgID, s, msg)
			return
		}
	}
	showBackupMenu(token, chat, msgID, s, "")
}

func showBackupMenu(token string, chat int64, msgID int, s install.BackupSchedule, status string) {
	ru := getLang() != "en"
	body := "🗓 <b>Расписание бэкапа</b>\n"
	body += "Сейчас: <code>" + install.FormatBackupSchedule(s) + "</code>\n"
	body += "По умолчанию ≈ 04:00 МСК (01:00 UTC)."
	if !ru {
		body = "🗓 <b>Backup schedule</b>\n"
		body += "Current: <code>" + install.FormatBackupSchedule(s) + "</code>\n"
		body += "Default ≈ 04:00 MSK (01:00 UTC)."
	}
	if status != "" {
		body = status + "\n\n" + body
	}
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("01:00 UTC", "m:backup:set:1:0", ""), btn("04:00 UTC", "m:backup:set:4:0", "")},
		{btn("22:00 UTC", "m:backup:set:22:0", ""), btn(map[bool]string{true: "Своё время", false: "Custom"}[ru], "m:backup:custom", "")},
		{btn(map[bool]string{true: "Запустить сейчас", false: "Run now"}[ru], "m:backup:run", "primary")},
		{btn(T("main_menu"), "m:menu", "")},
	}}
	reply(token, chat, msgID, body, kb)
}

func handleLocationCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	if data == "m:loc" {
		list, _ := sites.List()
		var b strings.Builder
		b.WriteString("📍 <b>Локации</b>\n")
		if !ru {
			b.WriteString("📍 <b>Locations</b>\n")
		}
		b.WriteString("<table bordered striped>\n<tr><th>#</th><th>id</th><th>name</th><th>kind</th></tr>\n")
		for i, s := range list {
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%s</td><td>%s</td></tr>\n", i+1, s.ID, esc(s.Name), esc(s.Kind)))
		}
		b.WriteString("</table>\n")
		if len(list) == 0 {
			b.WriteString("<i>Пока пусто. Добавьте: /loc add home home</i>\n")
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(map[bool]string{true: "Добавить home", false: "Add home"}[ru], "m:loc:add:home", "primary")},
			{btn(T("main_menu"), "m:menu", "")},
		}}
		reply(token, chat, msgID, b.String(), kb)
		return
	}
	if strings.HasPrefix(data, "m:loc:add:") {
		id := strings.TrimPrefix(data, "m:loc:add:")
		_, err := sites.Upsert(sites.Site{ID: id, Name: id, Kind: "home"})
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), mainKeyboard())
			return
		}
		handleLocationCB(token, chat, msgID, "m:loc")
	}
}

func handleQuotaCB(token string, chat int64, msgID int, data string) {
	// m:quota:user:50 or m:quota:menu
	parts := strings.Split(data, ":")
	if len(parts) >= 4 && parts[1] == "quota" {
		name := parts[2]
		gb, _ := strconv.ParseFloat(parts[3], 64)
		_ = vpn.SetSoftLimitGB(name, gb)
		reply(token, chat, msgID, fmt.Sprintf("✅ Soft limit <code>%s</code> = %.0f GiB", name, gb), userHubKeyboard(name))
		return
	}
}
