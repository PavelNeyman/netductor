package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
)

func handleEdgeGuestCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:edgeguest" && !strings.HasPrefix(data, "m:edgeguest:") {
		return false
	}
	ru := getLang() != "en"
	nav := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}

	if data == "m:edgeguest" {
		devs := edge.ListDevices()
		var ids []string
		var statuses []string
		for _, d := range devs {
			id := d.DeviceID
			if id == "" {
				continue
			}
			ids = append(ids, id)
			statuses = append(statuses, d.Status)
		}
		var b strings.Builder
		if ru {
			b.WriteString("📡 <b>Guest Wi‑Fi на роутере</b>\nВыберите устройство:\n")
		} else {
			b.WriteString("📡 <b>Router guest Wi‑Fi</b>\nPick a device:\n")
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
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeguest:dev:%s">%d</tg-button>`, id, i+1))
		}
		b.WriteString(`</tg-button-row>`)
		reply(token, chat, msgID, b.String(), nav)
		return true
	}
	rest := strings.TrimPrefix(data, "m:edgeguest:")
	if strings.HasPrefix(rest, "dev:") {
		id := strings.TrimPrefix(rest, "dev:")
		var b strings.Builder
		b.WriteString(fmt.Sprintf("📡 <b>%s</b>\n", esc(id)))
		stLabel, grLabel := "📊 Status", "⏱ Grant"
		if ru {
			stLabel, grLabel = "📊 Статус", "⏱ Выдать"
		}
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgeguest:st:%s">%s</tg-button>`, id, stLabel))
		b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="primary" data="m:edgeguest:g:%s">%s</tg-button>`, id, grLabel))
		b.WriteString(`</tg-button-row>`)
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("⬅️ Guest Wi‑Fi", "m:edgeguest", "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
		return true
	}
	if strings.HasPrefix(rest, "st:") {
		id := strings.TrimPrefix(rest, "st:")
		cid := edge.EnqueueCmd(id, "guest_status", "")
		if cid == "" {
			msg := "Enqueue failed (device not approved?)"
			if ru {
				msg = "Не удалось поставить в очередь (устройство не approved?)"
			}
			reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
				{btn("«", "m:edgeguest:dev:"+id, "primary"), btn(T("main_menu"), "m:menu", "")},
			}})
			return true
		}
		waitMsg := "⏳ Waiting for agent (up to ~90s)…"
		if ru {
			waitMsg = "⏳ Ждём ответ агента (до ~90с)…"
		}
		reply(token, chat, msgID, waitMsg+"\n<code>"+esc(cid)+"</code>", map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("«", "m:edgeguest:dev:"+id, "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
		go waitAndReplyEdgeCmd(token, chat, msgID, cid, id, ru)
		return true
	}
	if strings.HasPrefix(rest, "g:") {
		id := strings.TrimPrefix(rest, "g:")
		setState(chat, "wait_edgeguest_grant", id)
		msg := "Enter: <code>CODE minutes</code>, e.g. <code>7K2 10</code> (max 1440)."
		if ru {
			msg = "Введите: <code>КОД минуты</code>, например <code>7K2 10</code> (макс 1440)."
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("«", "m:edgeguest:dev:"+id, "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
		return true
	}
	return true
}

func handleEdgeGuestText(token string, chat int64, text string) bool {
	st, ok := chatState[chat]
	if !ok || st != "wait_edgeguest_grant" {
		return false
	}
	id := chatExtra[chat]
	setState(chat, "", "")
	ru := getLang() != "en"
	parts := strings.Fields(text)
	if len(parts) < 1 {
		return true
	}
	mins := 10
	code := parts[0]
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &mins)
	}
	if mins <= 0 {
		mins = 10
	}
	if mins > 1440 {
		mins = 1440
	}
	arg := fmt.Sprintf("%s|%d", code, mins)
	cid := edge.EnqueueCmd(id, "guest_grant", arg)
	if cid == "" {
		msg := "Enqueue failed (device not approved?)"
		if ru {
			msg = "Не удалось поставить в очередь"
		}
		reply(token, chat, 0, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:edgeguest:dev:"+id, "primary"), btn(T("main_menu"), "m:menu", "")}}})
		return true
	}
	waitMsg := fmt.Sprintf("⏳ Grant %s (%dm)… waiting agent", esc(code), mins)
	if ru {
		waitMsg = fmt.Sprintf("⏳ Grant %s (%dm)… ждём агент", esc(code), mins)
	}
	reply(token, chat, 0, waitMsg+"\n<code>"+esc(cid)+"</code>", map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:edgeguest:dev:"+id, "primary"), btn(T("main_menu"), "m:menu", "")}}})
	go waitAndReplyEdgeCmd(token, chat, 0, cid, id, ru)
	return true
}

func waitAndReplyEdgeCmd(token string, chat int64, msgID int, cid, deviceID string, ru bool) {
	res, err := edge.WaitCmdResult(cid, 90*time.Second)
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("«", "m:edgeguest:dev:"+deviceID, ""), btn(T("main_menu"), "m:menu", "primary")},
	}}
	if err != nil {
		msg := "⏱ Timeout: no agent result in 90s. Check online/heartbeat."
		if ru {
			msg = "⏱ Таймаут: агент не ответил за 90с. Проверьте online / heartbeat."
		}
		reply(token, chat, msgID, msg, kb)
		return
	}
	out, _ := res["output"].(string)
	if out == "" {
		out, _ = res["result"].(string)
	}
	if out == "" {
		out = fmt.Sprintf("%v", res)
	}
	title := "✅ Result"
	if ru {
		title = "✅ Результат"
	}
	body := title + "\n<pre>" + esc(truncateRunes(out, 3500)) + "</pre>"
	reply(token, chat, msgID, body, kb)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
