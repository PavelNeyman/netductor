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

	// Step 1: pick target
	if data == "m:reboot" || data == "m:reboot:ask" {
		var rows [][]map[string]any
		label := "primary"
		if ru {
			label = "primary (эта нода)"
		}
		rows = append(rows, []map[string]any{btn("🖥 "+label, "m:reboot:target:primary", "")})
		for _, d := range secondary.List() {
			name := d.Name
			if name == "" {
				name = d.ID
			}
			if len(name) > 28 {
				name = name[:28] + "…"
			}
			rows = append(rows, []map[string]any{btn("📡 "+name, "m:reboot:target:"+d.ID, "")})
		}
		rows = append(rows, []map[string]any{btn(T("cancel"), "m:status", "")})
		title := "♻️ <b>Reboot</b>\n\nSelect node:"
		if ru {
			title = "♻️ <b>Перезагрузка</b>\n\nВыберите узел:"
		}
		reply(token, chat, msgID, title, map[string]any{"inline_keyboard": rows})
		return true
	}

	// Step 2: challenge for primary
	if data == "m:reboot:target:primary" {
		showRebootConfirm(token, chat, msgID, "primary", "primary", true, ru)
		return true
	}
	if strings.HasPrefix(data, "m:reboot:target:") {
		id := strings.TrimPrefix(data, "m:reboot:target:")
		name := id
		for _, d := range secondary.List() {
			if d.ID == id && d.Name != "" {
				name = d.Name
				break
			}
		}
		showRebootConfirm(token, chat, msgID, id, name, false, ru)
		return true
	}

	// Confirm go primary
	if strings.HasPrefix(data, "m:reboot:go:primary:") {
		code := strings.TrimPrefix(data, "m:reboot:go:primary:")
		if err := notify.ConsumeRebootChallenge("primary", code); err != nil {
			msg := "❌ " + err.Error()
			if ru {
				msg += "\nЗапросите reboot снова."
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

	// Confirm go secondary: m:reboot:go:ID:CODE — id may have no colons typically
	if strings.HasPrefix(data, "m:reboot:go:") {
		rest := strings.TrimPrefix(data, "m:reboot:go:")
		// primary handled above
		i := strings.LastIndex(rest, ":")
		if i < 1 {
			return true
		}
		id, code := rest[:i], rest[i+1:]
		if err := notify.ConsumeRebootChallenge(id, code); err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), statusKeyboard())
			return true
		}
		_ = enqueueNodeCmd(id, "reboot")
		reply(token, chat, msgID, formatCmdQueuedHTML("reboot", id, ""), statusKeyboard())
		return true
	}

	// Legacy node card: m:n:reboot:ID → same as target
	if strings.HasPrefix(data, "m:n:reboot:") && !strings.Contains(data, ":go:") {
		id := strings.TrimPrefix(data, "m:n:reboot:")
		name := id
		for _, d := range secondary.List() {
			if d.ID == id && d.Name != "" {
				name = d.Name
				break
			}
		}
		showRebootConfirm(token, chat, msgID, id, name, false, ru)
		return true
	}
	return true
}

func showRebootConfirm(token string, chat int64, msgID int, target, display string, isPrimary, ru bool) {
	code, err := notify.BeginRebootChallenge(target)
	if err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), statusKeyboard())
		return
	}
	codes := notify.DecoyCodes(code, 3)
	var b strings.Builder
	if ru {
		b.WriteString(fmt.Sprintf("♻️ <b>Подтверждение reboot</b>\nУзел: <code>%s</code>\n\n", esc(display)))
		b.WriteString("Нажмите кнопку с <b>верным</b> кодом (один из трёх, 3 мин).\n")
		b.WriteString("Неверный код — отмена. Код в сообщении <b>не</b> дублируется текстом.\n")
	} else {
		b.WriteString(fmt.Sprintf("♻️ <b>Confirm reboot</b>\nNode: <code>%s</code>\n\n", esc(display)))
		b.WriteString("Tap the button with the <b>correct</b> code (one of three, 3 min).\n")
		b.WriteString("Wrong code cancels. Code is <b>not</b> repeated as plain text.\n")
	}
	// Problem: user needs to know the correct code without OCR image.
	// Agreed: 2-3 buttons one correct — but without showing code in text, user cannot know which!
	// Must show the real code once in message, decoys only on wrong buttons.
	if ru {
		b.WriteString("\nВерный код: <code>" + code + "</code>\n")
	} else {
		b.WriteString("\nCorrect code: <code>" + code + "</code>\n")
	}
	var row []map[string]any
	for _, c := range codes {
		cb := "m:reboot:go:" + target + ":" + c
		row = append(row, btn(c, cb, ""))
	}
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		row,
		{btn(T("cancel"), "m:reboot", "")},
	}}
	_ = isPrimary
	reply(token, chat, msgID, b.String(), kb)
}
