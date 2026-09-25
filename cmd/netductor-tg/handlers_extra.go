package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/dnsblock"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/sites"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleGuestCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	if data == "m:guest" {
		title := "⏱ <b>Гостевой доступ</b>\nОбщий пользователь <code>guest</code>. Выберите срок:"
		if !ru {
			title = "⏱ <b>Guest access</b>\nShared user <code>guest</code>. Choose TTL:"
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("10 мин", "m:guest:10m", ""), btn("1 час", "m:guest:1h", "")},
			{btn("24 ч", "m:guest:24h", ""), btn("7 дн", "m:guest:7d", "")},
			{btn(T("main_menu"), "m:menu", "primary")},
		}}
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
		body += "<pre><code>" + esc(link) + "</code></pre>"
	}
	dir := filepath.Join("/etc/netductor/clients", "guest")
	_ = os.MkdirAll(dir, 0o700)
	qrPath := filepath.Join(dir, "qr-vless.png")
	if link != "" {
		nl := string([]byte{10})
		html := `<img src="tg://photo?id=qr1"/>` + nl + body
		if p := ensureQRFile(qrPath, link); p != "" {
			replyRichWithPhoto(token, chat, msgID, html, p, "qr1", guestBackKB())
			return
		}
	}
	reply(token, chat, msgID, body, guestBackKB())
}

func guestBackKB() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⏱ Guest", "m:guest", ""), btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func handleDNSCB(token string, chat int64, msgID int, data string) {
	fmt.Fprintf(os.Stderr, "dns cb data=%q\n", data)
	if data == "m:dns:reload" {
		if err := dnsblock.ReloadBlocky(); err != nil {
			showDNSMenu(token, chat, msgID, "❌ Reload: "+err.Error())
			return
		}
		showDNSMenu(token, chat, msgID, "✅ Lists reloaded (blocky API)")
		return
	}
	if data == "m:dns" {
		showDNSMenu(token, chat, msgID, "")
		return
	}
	parts := strings.Split(data, ":")
	// m:dns:on:id or m:dns:off:id  (id may contain colons — join rest)
	if len(parts) >= 4 && parts[0] == "m" && parts[1] == "dns" && (parts[2] == "on" || parts[2] == "off") {
		on := parts[2] == "on"
		id := strings.Join(parts[3:], ":")
		if err := dnsblock.SetEnabled(id, on); err != nil {
			fmt.Fprintf(os.Stderr, "dns SetEnabled %s on=%v: %v\n", id, on, err)
			showDNSMenu(token, chat, msgID, "❌ "+err.Error())
			return
		}
		msg := "✅ " + id + " → "
		if on {
			msg += "ON (press 🔄 Reload to fetch)"
		} else {
			msg += "OFF"
		}
		showDNSMenu(token, chat, msgID, msg)
		return
	}
	showDNSMenu(token, chat, msgID, "")
}

func showDNSMenu(token string, chat int64, msgID int, status string) {
	body := dnsblock.FormatCatalogHTML()
	if status != "" {
		body = "<p>" + esc(status) + "</p>" + body
	}
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("back"), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	// Prefer in-place edit; delete+send only if rich edit fails.
	reply(token, chat, msgID, body, kb)
}

func handleBackupCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	s := install.LoadBackupSchedule()
	if data == "m:backup" {
		showBackupMenu(token, chat, msgID, s, "")
		return
	}
	if data == "m:backup:list" {
		names := install.ListBackupFiles()
		if len(names) > 12 {
			names = names[:12]
		}
		nl := string([]byte{10})
		var b strings.Builder
		b.WriteString("🗂 <b>Backups</b>" + nl)
		b.WriteString("<table bordered striped compact>" + nl)
		b.WriteString("<tr><th>#</th><th>file</th></tr>" + nl)
		for i, n := range names {
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>"+nl, i+1, n))
		}
		b.WriteString("</table>" + nl)
		const per = 4
		for i, n := range names {
			if i%per == 0 {
				if i > 0 {
					b.WriteString(`</tg-button-row>` + nl)
				}
				b.WriteString(`<tg-button-row align="left">`)
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:backup:restore:%s">%d</tg-button>`, n, i+1))
		}
		if len(names) > 0 {
			b.WriteString(`</tg-button-row>` + nl)
		}
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🗓 Backup", "m:backup", "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
		return
	}
	if data == "m:backup:keep" {
		setState(chat, "wait_backup_keep", "")
		reply(token, chat, msgID, "Keep last N backups (1–90), now: <code>"+strconv.Itoa(install.BackupKeepCount())+"</code>", map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🗓", "m:backup", ""), btn(T("main_menu"), "m:menu", "")},
		}})
		return
	}
	if strings.HasPrefix(data, "m:backup:restore:") {
		name := strings.TrimPrefix(data, "m:backup:restore:")
		path := filepath.Join("/var/lib/netductor/backups", name)
		reply(token, chat, msgID, "⏳ Restore <code>"+esc(name)+"</code>…", toolsKeyboard())
		err := install.Restore(path, "")
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), toolsKeyboard())
			return
		}
		reply(token, chat, msgID, "✅ Restore done: <code>"+esc(name)+"</code>", toolsKeyboard())
		return
	}
	if data == "m:backup:run" {
		showBackupMenu(token, chat, msgID, s, "⏳ Backup starting…")
		err := exec.Command("systemctl", "start", "netductor-backup.service").Start()
		if err != nil {
			showBackupMenu(token, chat, msgID, s, "❌ "+err.Error())
			return
		}
		// brief status
		time.Sleep(800 * time.Millisecond)
		out, _ := exec.Command("systemctl", "is-active", "netductor-backup.service").CombinedOutput()
		st := strings.TrimSpace(string(out))
		msg := "✅ systemctl start issued · unit=" + st
		if ru {
			msg = "✅ Бэкап запущен · unit=" + st
		}
		showBackupMenu(token, chat, msgID, s, msg)
		return
	}
	if data == "m:backup:custom" {
		setState(chat, "wait_backup_time", "")
		hint := "Введите время <code>HH:MM</code> (UTC), например <code>01:30</code>"
		if !ru {
			hint = "Enter <code>HH:MM</code> (UTC), e.g. <code>01:30</code>"
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
			showBackupMenu(token, chat, msgID, s, "✅ "+install.FormatBackupSchedule(s))
			return
		}
	}
	showBackupMenu(token, chat, msgID, s, "")
}

func showBackupMenu(token string, chat int64, msgID int, s install.BackupSchedule, status string) {
	ru := getLang() != "en"
	nl := "\n"
	var b strings.Builder
	if ru {
		b.WriteString("🗓 <b>Расписание бэкапа</b>" + nl)
	} else {
		b.WriteString("🗓 <b>Backup schedule</b>" + nl)
	}
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString("<tr><td>time</td><td><code>" + install.FormatBackupSchedule(s) + "</code></td></tr>" + nl)
	b.WriteString("<tr><td>default</td><td>01:00 UTC ≈ 04:00 MSK</td></tr>" + nl)
	b.WriteString("</table>" + nl)
	if status != "" {
		b.WriteString("<p>" + status + "</p>" + nl)
	}
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:backup:set:1:0">01:00</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:backup:set:4:0">04:00</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:backup:set:22:0">22:00</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:backup:custom">⏱</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="m:backup:run">▶</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:backup:list">📋</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:backup:keep">N</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("back"), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	reply(token, chat, msgID, b.String(), kb)
}

func handleLocationCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	if data == "m:loc" {
		list, _ := sites.List()
		var b strings.Builder
		if ru {
			b.WriteString("📍 <b>Локации</b>\n")
			b.WriteString("<i>Инвентарь мест: дом/офис. Сюда потом вешаются OpenWrt-агенты, шаблоны сети, статусы.</i>\n")
		} else {
			b.WriteString("📍 <b>Locations</b>\n")
			b.WriteString("<i>Place inventory: home/office. Bind OpenWrt agents, network templates, status here.</i>\n")
		}
		b.WriteString("<table bordered striped compact>\n<tr><th>#</th><th>name</th><th>kind</th><th>edges</th></tr>\n")
		for i, s := range list {
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><b>%s</b><br/><code>%s</code></td><td>%s</td><td>%d</td></tr>\n",
				i+1, esc(s.Name), s.ID, esc(s.Kind), len(s.EdgeIDs)))
		}
		b.WriteString("</table>\n")
		const per = 5
		for i, s := range list {
			if i%per == 0 {
				if i > 0 {
					b.WriteString(`</tg-button-row>` + "\n")
				}
				b.WriteString(`<tg-button-row align="left">`)
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:loc:open:%s">%d</tg-button>`, s.ID, i+1))
		}
		if len(list) > 0 {
			b.WriteString(`</tg-button-row>` + "\n")
		}
		if len(list) == 0 {
			if ru {
				b.WriteString("<i>Пусто — добавьте локацию.</i>\n")
			} else {
				b.WriteString("<i>Empty — add a location.</i>\n")
			}
		}
		addH, addO := "➕ Дом", "➕ Офис"
		if !ru {
			addH, addO = "➕ Home", "➕ Office"
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(addH, "m:loc:add:home", "primary"), btn(addO, "m:loc:add:office", "")},
			{btn(T("main_menu"), "m:menu", "")},
		}}
		reply(token, chat, msgID, b.String(), kb)
		return
	}
	if strings.HasPrefix(data, "m:loc:open:") {
		id := strings.TrimPrefix(data, "m:loc:open:")
		showLocationCard(token, chat, msgID, id)
		return
	}
	if strings.HasPrefix(data, "m:loc:del:") {
		id := strings.TrimPrefix(data, "m:loc:del:")
		_ = sites.Delete(id)
		handleLocationCB(token, chat, msgID, "m:loc")
		return
	}
	if strings.HasPrefix(data, "m:loc:rename:") {
		id := strings.TrimPrefix(data, "m:loc:rename:")
		setState(chat, "wait_loc_rename:"+id, "")
		reply(token, chat, msgID, "Новое имя для <code>"+esc(id)+"</code>:", map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("📍", "m:loc:open:"+id, ""), btn(T("main_menu"), "m:menu", "")},
		}})
		return
	}
	if strings.HasPrefix(data, "m:loc:add:") {
		kind := strings.TrimPrefix(data, "m:loc:add:")
		id := kind + "-" + strconv.FormatInt(time.Now().Unix()%100000, 10)
		name := "Home"
		if kind == "office" {
			name = "Office"
		}
		if kind != "home" && kind != "office" {
			name = kind
		}
		_, err := sites.Upsert(sites.Site{ID: id, Name: name, Kind: kind})
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), mainKeyboard())
			return
		}
		showLocationCard(token, chat, msgID, id)
		return
	}
}

func showLocationCard(token string, chat int64, msgID int, id string) {
	ru := getLang() != "en"
	s, ok := sites.Get(id)
	if !ok {
		reply(token, chat, msgID, "❌ not found", mainKeyboard())
		return
	}
	nl := "\n"
	var b strings.Builder
	b.WriteString("📍 <b>" + esc(s.Name) + "</b>" + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString("<tr><td>id</td><td><code>" + esc(s.ID) + "</code></td></tr>" + nl)
	b.WriteString("<tr><td>kind</td><td>" + esc(s.Kind) + "</td></tr>" + nl)
	b.WriteString("<tr><td>edges</td><td>" + strconv.Itoa(len(s.EdgeIDs)) + "</td></tr>" + nl)
	if s.Notes != "" {
		b.WriteString("<tr><td>notes</td><td>" + esc(s.Notes) + "</td></tr>" + nl)
	}
	b.WriteString("</table>" + nl)
	if len(s.EdgeIDs) > 0 {
		b.WriteString("<b>Edge IDs</b>\n<ul>\n")
		for _, e := range s.EdgeIDs {
			b.WriteString("<li><code>" + esc(e) + "</code></li>\n")
		}
		b.WriteString("</ul>\n")
	} else if ru {
		b.WriteString("<i>Агенты OpenWrt появятся после enroll и привязки к этой локации.</i>\n")
	} else {
		b.WriteString("<i>OpenWrt agents appear after enroll and bind to this location.</i>\n")
	}
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:loc:rename:` + id + `">✏️</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="m:loc:del:` + id + `">🗑</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("📍", "m:loc", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	reply(token, chat, msgID, b.String(), kb)
}

func handleQuotaCB(token string, chat int64, msgID int, data string) {
	parts := strings.Split(data, ":")
	// m:quota:name:50 | m:quota:name:custom
	if len(parts) >= 4 && parts[1] == "quota" {
		name := parts[2]
		if parts[3] == "custom" {
			setState(chat, "wait_quota:"+name, "")
			reply(token, chat, msgID, "Введите лимит в GiB (число) для <code>"+esc(name)+"</code>, или <code>0</code> = без лимита:", userHubKeyboard(name))
			return
		}
		gb, _ := strconv.ParseFloat(parts[3], 64)
		_ = vpn.SetSoftLimitGB(name, gb)
		reply(token, chat, msgID, formatUserHubHTML(name), userHubKeyboard(name))
		return
	}
}

func toolsHubHTML() string {
	ru := getLang() != "en"
	nl := "\n"
	if ru {
		return "🧰 <b>Инструменты</b>" + nl + "<i>Гость, DNS, бэкапы, локации, обновления.</i>"
	}
	return "🧰 <b>Tools</b>" + nl + "<i>Guest, DNS, backups, locations, updates.</i>"
}

func handleUpdatesCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	if data == "m:updates:self" {
		wait := "⏳ Updating from GitHub latest release…"
		if ru {
			wait = "⏳ Обновление с GitHub latest release…"
		}
		reply(token, chat, msgID, wait, toolsKeyboard())
		err := ndupdate.SelfReplace("netductor", "/usr/local/bin/netductor", "")
		_ = exec.Command("cp", "-f", "/usr/local/bin/netductor", "/opt/netductor/bin/netductor").Run()
		err2 := ndupdate.SelfReplace("tg", "/opt/netductor/bin/netductor-tg", "")
		_ = exec.Command("cp", "-f", "/opt/netductor/bin/netductor-tg", "/usr/local/bin/netductor-tg").Run()
		if tag, e := ndupdate.LatestReleaseTag(); e == nil {
			ndupdate.WriteVERSION(tag)
		}
		_ = exec.Command("systemctl", "restart", "netductor-api").Start()
		_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Start()
		msg := "✅ Updated from release; api + bot restarted"
		if ru {
			msg = "✅ Обновлено с release; api + bot перезапущены"
		}
		if err != nil || err2 != nil {
			msg = "❌ " + esc(fmt.Sprintf("netductor: %v; tg: %v", err, err2))
		}
		reply(token, chat, msgID, msg, toolsKeyboard())
		return
	}
	tag, err := ndupdate.LatestReleaseTag()
	local := "0.0.0"
	if b, e := os.ReadFile("/etc/netductor/VERSION"); e == nil {
		local = strings.TrimSpace(string(b))
	}
	var b strings.Builder
	if ru {
		b.WriteString("🔄 <b>Обновления</b>\n")
	} else {
		b.WriteString("🔄 <b>Updates</b>\n")
	}
	b.WriteString("<table bordered striped>\n<tr><th>item</th><th>value</th></tr>\n")
	b.WriteString("<tr><td>local</td><td><code>" + esc(local) + "</code></td></tr>\n")
	if err != nil {
		b.WriteString("<tr><td>latest</td><td>❌ " + esc(err.Error()) + "</td></tr>\n")
	} else {
		b.WriteString("<tr><td>latest</td><td><code>" + esc(tag) + "</code></td></tr>\n")
		if ndupdate.Newer(tag, local) {
			if ru {
				b.WriteString("<tr><td>status</td><td>🆕 доступно</td></tr>\n")
			} else {
				b.WriteString("<tr><td>status</td><td>🆕 available</td></tr>\n")
			}
		} else {
			if ru {
				b.WriteString("<tr><td>status</td><td>✅ актуально</td></tr>\n")
			} else {
				b.WriteString("<tr><td>status</td><td>✅ up to date</td></tr>\n")
			}
		}
	}
	b.WriteString("</table>\n")
	if ru {
		b.WriteString("<i>Агенты OpenWrt: точечное обновление через agent_update (без авто-раскатки).</i>\n")
	} else {
		b.WriteString("<i>OpenWrt agents: point update via agent_update (no auto-rollout).</i>\n")
	}
	upLabel := "⬆ Update primary"
	if ru {
		upLabel = "⬆ Обновить primary"
	}
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(upLabel, "m:updates:self", "primary")},
		{btn("🧰 Tools", "m:tools", ""), btn(T("main_menu"), "m:menu", "")},
	}}
	reply(token, chat, msgID, b.String(), kb)
}

