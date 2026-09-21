package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
)

func handleEdgeGuestCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:edgeguest" && !strings.HasPrefix(data, "m:edgeguest:") {
		return false
	}
	ru := getLang() != "en"
	if data == "m:edgeguest" {
		devs := edge.ListDevices()
		var rows [][]map[string]any
		for _, d := range devs {
			id := d.DeviceID
			if id == "" {
				continue
			}
			label := id
			if d.Status != "" {
				label = id + " · " + d.Status
			}
			rows = append(rows, []map[string]any{btn(label, "m:edgeguest:dev:"+id, "")})
		}
		if len(rows) == 0 {
			msg := "Нет edge-устройств"
			if !ru {
				msg = "No edge devices"
			}
			reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
				{btn(T("main_menu"), "m:menu", "primary")},
			}})
			return true
		}
		rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
		title := "📡 <b>Guest Wi‑Fi на роутере</b>\nВыберите устройство:"
		if !ru {
			title = "📡 <b>Router guest Wi‑Fi</b>\nPick a device:"
		}
		reply(token, chat, msgID, title, map[string]any{"inline_keyboard": rows})
		return true
	}
	rest := strings.TrimPrefix(data, "m:edgeguest:")
	if strings.HasPrefix(rest, "dev:") {
		id := strings.TrimPrefix(rest, "dev:")
		title := fmt.Sprintf("📡 <b>%s</b>", id)
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("📊 Status", "m:edgeguest:st:"+id, ""), btn("⏱ Grant", "m:edgeguest:g:"+id, "")},
			{btn("«", "m:edgeguest", ""), btn(T("main_menu"), "m:menu", "primary")},
		}}
		reply(token, chat, msgID, title, kb)
		return true
	}
	if strings.HasPrefix(rest, "st:") {
		id := strings.TrimPrefix(rest, "st:")
		cid := edge.EnqueueCmd(id, "guest_status", "")
		msg := fmt.Sprintf("Status queued <code>%s</code> (heartbeat ~30–90s).", esc(cid))
		if ru {
			msg = fmt.Sprintf("Status в очереди <code>%s</code> (heartbeat ~30–90с).", esc(cid))
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("«", "m:edgeguest:dev:"+id, "")},
		}})
		return true
	}
	if strings.HasPrefix(rest, "g:") {
		id := strings.TrimPrefix(rest, "g:")
		setState(chat, "wait_edgeguest_grant", id)
		msg := "Send: <code>CODE minutes</code> e.g. <code>7K2 10</code> (max 1440)."
		if ru {
			msg = "Введите: <code>КОД минуты</code>, например <code>7K2 10</code> (макс 1440)."
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("«", "m:edgeguest:dev:"+id, "")},
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
	reply(token, chat, 0, fmt.Sprintf("Grant queued <code>%s</code> → %s (%dm)", esc(cid), esc(code), mins),
		map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:edgeguest:dev:"+id, "")}}})
	return true
}
