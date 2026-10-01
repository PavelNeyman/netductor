package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
)

func handleEdgeLuciCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:edgeluci" && !strings.HasPrefix(data, "m:edgeluci:") {
		return false
	}
	ru := getLang() != "en"
	nav := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentRouters(), "m:cat:routers", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}

	if data == "m:edgeluci" {
		devs := edge.ListDevices()
		var ids []string
		var statuses []string
		for _, d := range devs {
			if d.DeviceID == "" {
				continue
			}
			ids = append(ids, d.DeviceID)
			statuses = append(statuses, d.Status)
		}
		var b strings.Builder
		if ru {
			b.WriteString("🔧 <b>LuCI на роутере</b>\nВкл/выкл без удаления. TTL по умолчанию 1ч.\nВыберите устройство:\n")
		} else {
			b.WriteString("🔧 <b>Router LuCI</b>\nEnable/disable (no uninstall). Default TTL 1h.\nPick a device:\n")
		}
		if len(ids) == 0 {
			if ru {
				b.WriteString("<i>Нет edge-устройств</i>")
			} else {
				b.WriteString("<i>No edge devices</i>")
			}
			reply(token, chat, msgID, b.String(), nav)
			return true
		}
		b.WriteString("<table bordered striped compact>\n<tr><th>#</th><th>id</th><th>st</th></tr>\n")
		for i, id := range ids {
			if i >= 12 {
				break
			}
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%s</td></tr>\n", i+1, esc(id), esc(statuses[i])))
		}
		b.WriteString("</table>\n")
		b.WriteString(`<tg-button-row align="left">`)
		for i, id := range ids {
			if i >= 12 {
				break
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:dev:%s">%d</tg-button>`, id, i+1))
		}
		b.WriteString(`</tg-button-row>`)
		reply(token, chat, msgID, b.String(), nav)
		return true
	}

	rest := strings.TrimPrefix(data, "m:edgeluci:")
	if strings.HasPrefix(rest, "dev:") {
		id := strings.TrimPrefix(rest, "dev:")
		var b strings.Builder
		b.WriteString(fmt.Sprintf("🔧 <b>%s</b>\n", esc(id)))
		en, dis, ext, st := "Enable 1h", "Disable", "Extend +24h", "Status"
		if ru {
			en, dis, ext, st = "Вкл 1ч", "Выкл", "Продлить +24ч", "Статус"
		}
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="primary" data="m:edgeluci:on:%s">%s</tg-button>`, id, en))
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="danger" data="m:edgeluci:off:%s">%s</tg-button>`, id, dis))
		b.WriteString(`</tg-button-row>`)
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:ext:%s">%s</tg-button>`, id, ext))
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:st:%s">%s</tg-button>`, id, st))
		b.WriteString(`</tg-button-row>`)
		b.WriteString(`<tg-button-row align="left">`)
		if ru {
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:on4:%s">4ч</tg-button>`, id))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:on24:%s">24ч</tg-button>`, id))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:on72:%s">72ч</tg-button>`, id))
		} else {
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:on4:%s">4h</tg-button>`, id))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:on24:%s">24h</tg-button>`, id))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeluci:on72:%s">72h</tg-button>`, id))
		}
		b.WriteString(`</tg-button-row>`)
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("«", "m:edgeluci", "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
		return true
	}

	run := func(id, action, arg string) {
		cid := edge.EnqueueCmd(id, action, arg)
		back := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("«", "m:edgeluci:dev:"+id, "primary"), btn(T("main_menu"), "m:menu", "")},
		}}
		if cid == "" {
			msg := "❌ enqueue failed (device not approved?)"
			if ru {
				msg = "❌ очередь пуста (устройство не approved?)"
			}
			reply(token, chat, msgID, msg, back)
			return
		}
		wait := "⏳ …"
		if ru {
			wait = "⏳ ждём агент…"
		}
		reply(token, chat, msgID, wait+"\n<code>"+esc(cid)+"</code>", back)
		go func() {
			res, err := edge.WaitCmdResult(cid, 90*time.Second)
			var body string
			if err != nil {
				body = "❌ " + esc(err.Error())
			} else {
				rtext, _ := res["result"].(string)
				if rtext == "" {
					rtext = fmt.Sprint(res)
				}
				body = "✅ <pre>" + esc(rtext) + "</pre>"
				if strings.Contains(action, "enable") || strings.Contains(action, "extend") {
					body = "🔧 LuCI <code>" + esc(id) + "</code>\n" + body
				}
			}
			reply(token, chat, 0, body, back)
		}()
	}

	switch {
	case strings.HasPrefix(rest, "st:"):
		run(strings.TrimPrefix(rest, "st:"), "luci_status", "")
	case strings.HasPrefix(rest, "off:"):
		run(strings.TrimPrefix(rest, "off:"), "luci_disable", "")
	case strings.HasPrefix(rest, "ext:"):
		run(strings.TrimPrefix(rest, "ext:"), "luci_extend", "hours=24")
	case strings.HasPrefix(rest, "on4:"):
		run(strings.TrimPrefix(rest, "on4:"), "luci_enable", "hours=4")
	case strings.HasPrefix(rest, "on24:"):
		run(strings.TrimPrefix(rest, "on24:"), "luci_enable", "hours=24")
	case strings.HasPrefix(rest, "on72:"):
		run(strings.TrimPrefix(rest, "on72:"), "luci_enable", "hours=72")
	case strings.HasPrefix(rest, "on:"):
		run(strings.TrimPrefix(rest, "on:"), "luci_enable", "hours=1")
	default:
		reply(token, chat, msgID, "?", nav)
	}
	return true
}
