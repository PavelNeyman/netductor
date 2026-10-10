package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func handleRebootCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:reboot" && !strings.HasPrefix(data, "m:reboot:") && !strings.HasPrefix(data, "m:n:reboot:") {
		return false
	}
	ru := getLang() != "en"

	// Primary host: m:reboot → ask, m:reboot:go:CODE → consume
	if data == "m:reboot" || data == "m:reboot:ask" {
		code, err := notify.BeginRebootChallenge("primary")
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), statusKeyboard())
			return true
		}
		var b strings.Builder
		if ru {
			b.WriteString("♻️ <b>Перезагрузка primary</b>\n\n")
			b.WriteString("Код подтверждения (3 мин): <code>" + code + "</code>\n")
			b.WriteString("Нажмите кнопку с <b>этим</b> кодом. Случайный/чужой reboot без кода не пройдёт.\n")
		} else {
			b.WriteString("♻️ <b>Reboot primary</b>\n\n")
			b.WriteString("Confirmation code (3 min): <code>" + code + "</code>\n")
			b.WriteString("Tap the button with <b>this</b> code. Random reboot without code will not run.\n")
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("✅ "+code, "m:reboot:go:"+code, "")},
			{btn(T("cancel"), "m:status", "")},
		}}
		reply(token, chat, msgID, b.String(), kb)
		return true
	}
	if strings.HasPrefix(data, "m:reboot:go:") {
		code := strings.TrimPrefix(data, "m:reboot:go:")
		if err := notify.ConsumeRebootChallenge("primary", code); err != nil {
			msg := "❌ " + err.Error()
			if ru {
				msg = "❌ " + err.Error() + "\nЗапросите reboot снова."
			}
			reply(token, chat, msgID, msg, statusKeyboard())
			return true
		}
		reply(token, chat, msgID, T("reboot_queued")+"\n<code>primary</code>", statusKeyboard())
		go func() {
			_ = exec.Command("shutdown", "-r", "+1", "netductor TG reboot").Run()
		}()
		return true
	}

	// Secondary: m:n:reboot:ID → ask, m:n:reboot:go:ID:CODE
	if strings.HasPrefix(data, "m:n:reboot:go:") {
		rest := strings.TrimPrefix(data, "m:n:reboot:go:")
		// id may contain colons — code is last :part
		i := strings.LastIndex(rest, ":")
		if i < 1 {
			return true
		}
		id, code := rest[:i], rest[i+1:]
		if err := notify.ConsumeRebootChallenge(id, code); err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), nodeCardKeyboard(id))
			return true
		}
		_ = enqueueNodeCmd(id, "reboot")
		reply(token, chat, msgID, formatCmdQueuedHTML("reboot", id, ""), nodeCardKeyboard(id))
		return true
	}
	if strings.HasPrefix(data, "m:n:reboot:") {
		id := strings.TrimPrefix(data, "m:n:reboot:")
		code, err := notify.BeginRebootChallenge(id)
		if err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), nodeCardKeyboard(id))
			return true
		}
		name := id
		for _, d := range secondary.List() {
			if d.ID == id && d.Name != "" {
				name = d.Name
				break
			}
		}
		var b strings.Builder
		if ru {
			b.WriteString(fmt.Sprintf("♻️ <b>Перезагрузка secondary</b> <code>%s</code>\n\nКод: <code>%s</code> (3 мин)\n", esc(name), code))
		} else {
			b.WriteString(fmt.Sprintf("♻️ <b>Reboot secondary</b> <code>%s</code>\n\nCode: <code>%s</code> (3 min)\n", esc(name), code))
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("✅ "+code, "m:n:reboot:go:"+id+":"+code, "")},
			{btn(T("cancel"), "m:n:o:"+id, "")},
		}}
		reply(token, chat, msgID, b.String(), kb)
		return true
	}
	return true
}
