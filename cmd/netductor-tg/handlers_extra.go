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
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/sites"
	ndstack "github.com/PavelNeyman/netductor/internal/stack"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleGuestCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	if data == "m:guest" {
		title := "⏱ <b>Гостевой доступ</b>\nОбщий пользователь <code>guest</code>. Выберите срок:\n"
		if !ru {
			title = "⏱ <b>Guest access</b>\nShared user <code>guest</code>. Choose TTL:\n"
		}
		if ru {
			title += `<tg-button-row align="left"><tg-button type="callback_data" style="link" data="m:guest:10m">10 мин</tg-button><tg-button type="callback_data" style="link" data="m:guest:1h">1 ч</tg-button><tg-button type="callback_data" style="link" data="m:guest:24h">24 ч</tg-button><tg-button type="callback_data" style="link" data="m:guest:7d">7 дн</tg-button></tg-button-row>`
		} else {
			title += `<tg-button-row align="left"><tg-button type="callback_data" style="link" data="m:guest:10m">10m</tg-button><tg-button type="callback_data" style="link" data="m:guest:1h">1h</tg-button><tg-button type="callback_data" style="link" data="m:guest:24h">24h</tg-button><tg-button type="callback_data" style="link" data="m:guest:7d">7d</tg-button></tg-button-row>`
		}
		reply(token, chat, msgID, title, navKeyboard("m:tools", parentTools()))
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
	return navKeyboard("m:tools", parentTools())
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
		if getLang() != "en" {
			if on {
				msg += "ВКЛ (нажмите 🔄 Reload чтобы скачать)"
			} else {
				msg += "ВЫКЛ"
			}
		} else if on {
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
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
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
		if ru {
			b.WriteString("🗂 <b>Бэкапы</b>" + nl)
		} else {
			b.WriteString("🗂 <b>Backups</b>" + nl)
		}
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
		keepMsg := "Keep last N backups (1–90), now: <code>" + strconv.Itoa(install.BackupKeepCount()) + "</code>"
		if ru {
			keepMsg = "Хранить последних N бэкапов (1–90), сейчас: <code>" + strconv.Itoa(install.BackupKeepCount()) + "</code>"
		}
		reply(token, chat, msgID, keepMsg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🗓", "m:backup", ""), btn(T("main_menu"), "m:menu", "")},
		}})
		return
	}
	if strings.HasPrefix(data, "m:backup:restore:") {
		name := strings.TrimPrefix(data, "m:backup:restore:")
		path := filepath.Join("/var/lib/netductor/backups", name)
		rst := "⏳ Restore <code>" + esc(name) + "</code>…"
		if ru {
			rst = "⏳ Восстановление <code>" + esc(name) + "</code>…"
		}
		reply(token, chat, msgID, rst, navKeyboard("m:tools", parentTools()))
		err := install.Restore(path, "")
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:tools", parentTools()))
			return
		}
		done := "✅ Restore done: <code>" + esc(name) + "</code>"
		if ru {
			done = "✅ Восстановлено: <code>" + esc(name) + "</code>"
		}
		reply(token, chat, msgID, done, navKeyboard("m:tools", parentTools()))
		return
	}
	if data == "m:backup:run" {
		st0 := "⏳ Backup starting…"
		if ru {
			st0 = "⏳ Запуск бэкапа…"
		}
		showBackupMenu(token, chat, msgID, s, st0)
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
	if ru {
		b.WriteString("<tr><th>поле</th><th>значение</th></tr>" + nl)
		b.WriteString("<tr><td>время</td><td><code>" + install.FormatBackupSchedule(s) + "</code></td></tr>" + nl)
		b.WriteString("<tr><td>по умолчанию</td><td>01:00 UTC ≈ 04:00 MSK</td></tr>" + nl)
	} else {
		b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
		b.WriteString("<tr><td>time</td><td><code>" + install.FormatBackupSchedule(s) + "</code></td></tr>" + nl)
		b.WriteString("<tr><td>default</td><td>01:00 UTC ≈ 04:00 MSK</td></tr>" + nl)
	}
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
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
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
		if ru {
			b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" style="primary" data="m:loc:add:home">➕ Дом</tg-button><tg-button type="callback_data" style="link" data="m:loc:add:office">➕ Офис</tg-button></tg-button-row>`)
		} else {
			b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" style="primary" data="m:loc:add:home">➕ Home</tg-button><tg-button type="callback_data" style="link" data="m:loc:add:office">➕ Office</tg-button></tg-button-row>`)
		}
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(T("main_menu"), "m:menu", "primary")},
		}})
		return
	}
	
	
	if strings.HasPrefix(data, "m:loc:roomadd:") {
		siteID := strings.TrimPrefix(data, "m:loc:roomadd:")
		setState(chat, "wait_room_id", siteID)
		chatExtra[chat] = siteID
		msg := "Room id (a-z0-9-, max 32), e.g. kitchen"
		if getLang() != "en" {
			msg = "Id комнаты (a-z0-9-, до 32), например kitchen"
		}
		reply(token, chat, msgID, msg, navKeyboard("m:loc:rooms:"+siteID, "Rooms"))
		return
	}
	if strings.HasPrefix(data, "m:loc:rooms:") {
		siteID := strings.TrimPrefix(data, "m:loc:rooms:")
		showRoomsList(token, chat, msgID, siteID)
		return
	}
	if strings.HasPrefix(data, "m:loc:room:") {
		rest := strings.TrimPrefix(data, "m:loc:room:")
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) != 2 {
			reply(token, chat, msgID, "bad room", mainKeyboard())
			return
		}
		showRoomCard(token, chat, msgID, parts[0], parts[1])
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
		rnMsg := "Новое имя для <code>" + esc(id) + "</code>:"
		if !ru {
			rnMsg = "New name for <code>" + esc(id) + "</code>:"
		}
		pl := "Locations"
		if ru {
			pl = "Локации"
		}
		reply(token, chat, msgID, rnMsg, navKeyboard("m:loc", pl))
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
	if ru {
		b.WriteString("<i>✏️ переименовать · 🗑 удалить</i>" + nl)
	} else {
		b.WriteString("<i>✏️ rename · 🗑 delete</i>" + nl)
	}
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="m:loc:rooms:` + id + `">🚪 Rooms</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:loc:rename:` + id + `">✏️</tg-button>`)
	b.WriteString(`<tg-button type="callback_data" style="danger" data="m:loc:del:` + id + `">🗑</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	pl := "Locations"
	if ru {
		pl = "Локации"
	}
	reply(token, chat, msgID, b.String(), navKeyboard("m:loc", pl))
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

func handleUpdatesCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	// m:updates:token:set|clear
	if data == "m:updates:token:set" || data == "m:updates:token" {
		st := ndupdate.GetTokensStatus()
		nl := string([]byte{10})
		msg := "🔑 <b>GitHub tokens</b>" + nl
		msg += "releases: "
		if st.Releases.Configured {
			msg += "✅ <code>" + esc(st.Releases.Hint) + "</code>"
		} else {
			msg += "❌"
		}
		msg += nl + "repos: "
		if st.Repos.Configured {
			msg += "✅ <code>" + esc(st.Repos.Hint) + "</code>"
		} else {
			msg += "❌ (fallback to releases if set)"
		}
		msg += nl + nl + "Pick which token to set:"
		if ru {
			msg = "🔑 <b>GitHub tokens</b>" + nl
			msg += "releases: "
			if st.Releases.Configured {
				msg += "✅ <code>" + esc(st.Releases.Hint) + "</code>"
			} else {
				msg += "❌"
			}
			msg += nl + "repos: "
			if st.Repos.Configured {
				msg += "✅ <code>" + esc(st.Repos.Hint) + "</code>"
			} else {
				msg += "❌ (иначе = releases)"
			}
			msg += nl + nl + "Какой токен задать:"
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("Releases API", "m:updates:token:set:releases", "primary"), btn("Repos mirror", "m:updates:token:set:repos", "")},
			{btn("Clear releases", "m:updates:token:clear:releases", ""), btn("Clear repos", "m:updates:token:clear:repos", "")},
			{btn("⬅️", "m:updates", "primary"), btn(T("main_menu"), "m:menu", "")},
		}}
		reply(token, chat, msgID, msg, kb)
		return
	}
	if strings.HasPrefix(data, "m:updates:token:set:") {
		kind := strings.TrimPrefix(data, "m:updates:token:set:")
		setState(chat, "wait_github_token:"+kind, "")
		msg := "🔑 Send PAT for <b>" + esc(kind) + "</b>\n<code>/cancel</code>"
		if ru {
			msg = "🔑 Пришлите PAT для <b>" + esc(kind) + "</b>\n<code>/cancel</code>"
		}
		reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
		return
	}
	if strings.HasPrefix(data, "m:updates:token:clear:") {
		kind := strings.TrimPrefix(data, "m:updates:token:clear:")
		_ = ndupdate.ClearTokenKind(kind)
		msg := "✅ cleared " + kind
		reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
		return
	}
	if data == "m:updates:token:clear" {
		_ = ndupdate.ClearTokenKind(ndupdate.TokenKindReleases)
		msg := "✅ GitHub releases token cleared"
		if ru {
			msg = "✅ GitHub releases token удалён"
		}
		reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
		return
	}

	// m:updates:edge:<device_id> — enqueue agent_update with SHA from release SHA256SUMS (R8)
	if strings.HasPrefix(data, "m:updates:edge:") {
		did := strings.TrimPrefix(data, "m:updates:edge:")
		d, ok := edge.GetDevice(did)
		if !ok {
			msg := "❌ device not found: " + esc(did)
			if ru {
				msg = "❌ устройство не найдено: " + esc(did)
			}
			reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
			return
		}
		arch := strings.TrimSpace(d.Arch)
		if arch == "" && d.Extra != nil {
			if v, ok := d.Extra["arch"].(string); ok {
				arch = strings.TrimSpace(v)
			}
			if arch == "" {
				if v, ok := d.Extra["goarch"].(string); ok {
					arch = strings.TrimSpace(v)
				}
			}
		}
		if arch == "" {
			msg := "⚠️ no arch for <code>" + esc(did) + "</code> yet — wait for agent heartbeat (arch in metrics), then retry."
			if ru {
				msg = "⚠️ нет arch для <code>" + esc(did) + "</code> — дождитесь heartbeat агента (поле arch), затем снова."
			}
			reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
			return
		}
		tag, err := ndupdate.LatestReleaseTag()
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:updates", parentTools()))
			return
		}
		arg, err := ndupdate.AgentUpdateArg(tag, arch)
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:updates", parentTools()))
			return
		}
		cid := edge.EnqueueCmd(did, "agent_update", arg)
		if cid == "" {
			msg := "❌ enqueue failed (not approved?)"
			if ru {
				msg = "❌ enqueue не удался (не approved?)"
			}
			reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
			return
		}
		msg := "✅ agent_update queued\n<code>" + esc(did) + "</code> " + esc(arch) + " → " + esc(tag) + "\ncmd " + esc(cid)
		if ru {
			msg = "✅ agent_update в очереди\n<code>" + esc(did) + "</code> " + esc(arch) + " → " + esc(tag) + "\ncmd " + esc(cid)
		}
		reply(token, chat, msgID, msg, navKeyboard("m:updates", parentTools()))
		return
	}
	// m:updates:apply:v0.9.97 or m:updates:self (latest)
	if data == "m:updates:self" || strings.HasPrefix(data, "m:updates:apply:") {
		tag := ""
		if strings.HasPrefix(data, "m:updates:apply:") {
			tag = strings.TrimPrefix(data, "m:updates:apply:")
		}
		wait := "⏳ Backup + update…"
		if ru {
			wait = "⏳ Бэкап + обновление…"
		}
		reply(token, chat, msgID, wait, navKeyboard("m:tools", parentTools()))
		// Never run backup/download inside the bot process (OOM: tar+encrypt in RAM).
		// Delegate to node CLI which is a separate process with its own memory limit.
		if tag == "" {
			var err error
			tag, err = ndupdate.LatestReleaseTag()
			if err != nil {
				reply(token, chat, msgID, "❌ "+esc(err.Error()), navKeyboard("m:tools", parentTools()))
				return
			}
		}
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag
		}
		// Ignore double-taps while apply lock held or oneshot still scheduled.
		if busy, cur, _ := ndstack.ApplyInProgress(); busy {
			msg := ndstack.ApplyLockStatusLine(ru)
			if msg == "" {
				msg = "⏳ busy: " + esc(cur)
			}
			clr := "🔓 Clear lock"
			if ru {
				clr = "🔓 Сбросить lock"
			}
			rows := [][]map[string]any{
				{btnDisabled("⏳ " + cur)},
				{btn(clr, "m:updates:unlock", "danger"), btn("⬅️ "+parentTools(), "m:tools", "primary")},
			}
			reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": rows})
			return
		}
		// Refuse if previous systemd unit still active
		if out, _ := exec.Command("systemctl", "is-active", "netductor-stack-apply.service").Output(); strings.TrimSpace(string(out)) == "activating" || strings.TrimSpace(string(out)) == "active" {
			msg := "⏳ Update job still running"
			if ru {
				msg = "⏳ Задача обновления ещё выполняется"
			}
			reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btnDisabled("⏳ …"), btn("⬅️ "+parentTools(), "m:tools", "primary")}}})
			return
		}
		// Schedule apply outside bot process. stack apply restarts units itself — do not double-restart.
		// Unified scheduler (same as API) — avoids double lock / divergent paths.
		runErr := exec.Command("/usr/local/bin/netductor", "stack", "schedule", tag).Run()
		var msg string
		if ru {
			msg = "⏳ <b>Обновление запланировано</b> <code>" + esc(tag) + "</code>\n"
			msg += "<i>Бот перезапустится через ~15–30 с. Не жмите Update повторно.</i>\n"
			msg += "Лог: <code>/var/log/netductor-stack-apply.log</code>"
		} else {
			msg = "⏳ <b>Update scheduled</b> <code>" + esc(tag) + "</code>\n"
			msg += "<i>Bot will restart in ~15–30s. Do not tap Update again.</i>\n"
			msg += "Log: <code>/var/log/netductor-stack-apply.log</code>"
		}
		if runErr != nil {
			msg = "❌ schedule: " + esc(runErr.Error())
		}
		reply(token, chat, msgID, msg, navKeyboard("m:tools", parentTools()))
		return
	}
	tag, err := ndupdate.LatestReleaseTag()
	local := ndver.Release
	if b, e := os.ReadFile("/etc/netductor/VERSION"); e == nil && strings.TrimSpace(string(b)) != "" {
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
	ts := ndupdate.GetTokenStatus()
	tokCell := "❌"
	if ts.Configured {
		tokCell = "✅ " + esc(ts.Hint) + " <i>(" + esc(ts.Source) + ")</i>"
	}
	b.WriteString("<tr><td>github token</td><td>" + tokCell + "</td></tr>\n")
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
		b.WriteString("<i>Выберите релиз. Перед apply — backup + ожидание backup_pull. Edge: agent_update точечно (без авто-раскатки).</i>\n")
	} else {
		b.WriteString("<i>Pick a release. Pre-apply backup + wait backup_pull. Edge: point agent_update (no auto-rollout).</i>\n")
	}
	applyBusy, _, _ := ndstack.ApplyInProgress()
	if line := ndstack.ApplyLockStatusLine(ru); line != "" {
		b.WriteString(line + "\n")
	}
	// release picker (top 6)
	rows := [][]map[string]any{}
	if list, e := ndupdate.ListReleases(6); e == nil {
		for _, r := range list {
			label := r.Tag
			if r.Tag == tag {
				if ru {
					label = r.Tag + " (latest)"
				} else {
					label = r.Tag + " (latest)"
				}
			}
			if applyBusy {
				rows = append(rows, []map[string]any{btnDisabled("⏳ " + label)})
			} else {
				rows = append(rows, []map[string]any{btn("⬆ "+label, "m:updates:apply:"+r.Tag, "primary")})
			}
		}
	} else {
		// API rate-limited: still offer latest attempt + current local tag
		if ru {
			b.WriteString("<i>Список релизов недоступен (GitHub 403). Попробуйте позже или задайте NETDUCTOR_GITHUB_TOKEN на primary.</i>\n")
		} else {
			b.WriteString("<i>Release list unavailable (GitHub 403). Retry later or set NETDUCTOR_GITHUB_TOKEN on primary.</i>\n")
		}
		if applyBusy {
			rows = append(rows, []map[string]any{btnDisabled("⏳ Latest")})
		} else {
			rows = append(rows, []map[string]any{btn("⬆ Latest", "m:updates:self", "primary")})
		}
		if local != "" {
			rows = append(rows, []map[string]any{btn("⬆ v"+strings.TrimPrefix(local, "v"), "m:updates:apply:v"+strings.TrimPrefix(local, "v"), "")})
		}
	}
	if applyBusy {
		clr := "🔓 Clear lock"
		if ru {
			clr = "🔓 Сбросить lock"
		}
		rows = append(rows, []map[string]any{btn(clr, "m:updates:unlock", "danger")})
	}
	rows = append(rows, []map[string]any{
		btn("🔑 GitHub token", "m:updates:token:set", "primary"),
		btn("🗑 Clear token", "m:updates:token:clear", "danger"),
	})
	// edge agent_update (point)
	for _, d := range edge.ListDevices() {
		if d.DeviceID == "" {
			continue
		}
		lab := "edge " + d.DeviceID
		if d.Hostname != "" {
			lab = d.Hostname
		}
		ver := d.Agent
		if ver == "" {
			ver = "?"
		}
		if ru {
			rows = append(rows, []map[string]any{btn("📡 "+lab+" ("+ver+") ⬆", "m:updates:edge:"+d.DeviceID, "")})
		} else {
			rows = append(rows, []map[string]any{btn("📡 "+lab+" ("+ver+") ⬆", "m:updates:edge:"+d.DeviceID, "")})
		}
	}
	rows = append(rows, []map[string]any{
		btn("⬅️ "+parentTools(), "m:tools", "primary"),
		btn(T("main_menu"), "m:menu", ""),
	})
	reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
}

func trimRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}


func showRoomsList(token string, chat int64, msgID int, siteID string) {
	ru := getLang() != "en"
	list, err := sites.ListRooms(siteID)
	nl := "\n"
	var b strings.Builder
	if ru {
		b.WriteString("🚪 <b>Комнаты</b> · <code>" + esc(siteID) + "</code>" + nl)
	} else {
		b.WriteString("🚪 <b>Rooms</b> · <code>" + esc(siteID) + "</code>" + nl)
	}
	if err != nil {
		b.WriteString(esc(err.Error()) + nl)
	}
	if len(list) == 0 {
		if ru {
			b.WriteString("<i>Пусто — добавьте через CLI/Web: sites rooms add</i>" + nl)
		} else {
			b.WriteString("<i>Empty — add via CLI/Web: sites rooms add</i>" + nl)
		}
	}
	for i, r := range list {
		if i >= 12 {
			break
		}
		label := r.Name
		if label == "" {
			label = r.ID
		}
		label = fmt.Sprintf("%s · %d📷", label, r.PhotoCount)
		b.WriteString(fmt.Sprintf(`<tg-button-row align="left"><tg-button type="callback_data" style="primary" data="m:loc:room:%s:%s">%s</tg-button></tg-button-row>`+nl, siteID, r.ID, esc(label)))
	}
	b.WriteString(`<tg-button-row align="left"><tg-button type="callback_data" style="primary" data="m:loc:roomadd:` + siteID + `">➕</tg-button></tg-button-row>` + nl)
	reply(token, chat, msgID, b.String(), navKeyboard("m:loc:open:"+siteID, siteID))
}

func showRoomCard(token string, chat int64, msgID int, siteID, roomID string) {
	ru := getLang() != "en"
	r, ok := sites.GetRoom(siteID, roomID)
	if !ok {
		reply(token, chat, msgID, "room not found", navKeyboard("m:loc:rooms:"+siteID, "Rooms"))
		return
	}
	nl := "\n"
	var b strings.Builder
	b.WriteString("🚪 <b>" + esc(r.Name) + "</b>" + nl)
	b.WriteString("<code>" + esc(r.ID) + "</code> · photos " + strconv.Itoa(r.PhotoCount) + "/" + strconv.Itoa(sites.MaxRoomPhotos) + nl)
	if r.Description != "" {
		b.WriteString(esc(r.Description) + nl)
	}
	if len(r.CameraIDs) > 0 {
		b.WriteString("<b>cams</b> " + esc(strings.Join(r.CameraIDs, ", ")) + nl)
	}
	if r.Notes != "" {
		b.WriteString(esc(r.Notes) + nl)
	}
	// send photos as separate messages (max 5)
	for i := 0; i < r.PhotoCount && i < sites.MaxRoomPhotos; i++ {
		data, err := sites.GetPhoto(siteID, roomID, i)
		if err != nil {
			continue
		}
		_ = sendPhotoBytes(token, chat, data, fmt.Sprintf("%s/%s #%d", siteID, roomID, i))
	}
	b.WriteString(`<tg-button-row align="left">`)
	b.WriteString(`<tg-button type="callback_data" style="link" data="m:loc:rooms:` + siteID + `">⬅️</tg-button>`)
	b.WriteString(`</tg-button-row>` + nl)
	_ = ru
	reply(token, chat, msgID, b.String(), navKeyboard("m:loc:rooms:"+siteID, "Rooms"))
}
