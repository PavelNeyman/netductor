package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/stack"
)

func handleStackCB(token string, chat int64, msgID int, data string) {
	ru := getLang() != "en"
	kb := func() map[string]any {
		rb := "↩️ Rollback"
		if ru {
			rb = "↩️ Откат"
		}
		return map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🔄", "m:stack", "primary"), btn("🩹 Heal", "m:stack:heal", ""), btn("🔧 WD", "m:stack:watchdog", "")},
			{btn(rb, "m:stack:rollback", ""), btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
		}}
	}
	switch {
	case data == "m:stack":
		st := stack.Collect()
		html := stack.FormatHTML(st)
		if ru {
			html += "\n\n<i>Откат — last-good (после успешного apply = текущий релиз). Нужно подтверждение.</i>"
		} else {
			html += "\n\n<i>Rollback — last-good (after successful apply = current). Requires confirm.</i>"
		}
		reply(token, chat, msgID, html, kb())
	case data == "m:stack:heal":
		out, err := exec.Command("/usr/local/bin/netductor", "stack", "heal").CombinedOutput()
		msg := "🩹 Heal\n<pre>" + esc(trimRunes(string(out), 1200)) + "</pre>"
		if err != nil {
			msg = "❌ " + esc(fmt.Sprintf("%v\n%s", err, trimRunes(string(out), 800)))
		}
		reply(token, chat, msgID, msg+"\n\n"+stack.FormatHTML(stack.Collect()), kb())
	case data == "m:stack:rollback":
		msg := "↩️ <b>Rollback last-good?</b>\n<i>After a successful update, prev is the same release — safe. Ancient 0.9.121 is no longer kept as last-good.</i>"
		yes, cancel := "✅ Confirm", "⬅️ Cancel"
		if ru {
			msg = "↩️ <b>Откатить на last-good?</b>\n<i>После успешного обновления prev = текущий релиз. Древний 0.9.121 больше не хранится как last-good.</i>"
			yes, cancel = "✅ Подтвердить", "⬅️ Отмена"
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(yes, "m:stack:rollback:yes", "danger"), btn(cancel, "m:stack", "")},
		}})
	case data == "m:stack:rollback:yes":
		wait := "⏳ Rollback…"
		if ru {
			wait = "⏳ Откат…"
		}
		reply(token, chat, msgID, wait, kb())
		out, err := exec.Command("/usr/local/bin/netductor", "stack", "rollback").CombinedOutput()
		msg := "✅ Rollback\n<pre>" + esc(trimRunes(string(out), 1200)) + "</pre>"
		if err != nil {
			msg = "❌ " + esc(fmt.Sprintf("%v\n%s", err, trimRunes(string(out), 1200)))
		}
		reply(token, chat, msgID, msg+"\n\n"+stack.FormatHTML(stack.Collect()), kb())
	case data == "m:stack:watchdog":
		out, err := exec.Command("/usr/local/bin/netductor", "stack", "watchdog").CombinedOutput()
		msg := "🔧 Watchdog\n<pre>" + esc(trimRunes(string(out), 1200)) + "</pre>"
		if err != nil {
			msg = "❌ " + esc(err.Error())
		}
		reply(token, chat, msgID, msg, kb())
	default:
		reply(token, chat, msgID, stack.FormatHTML(stack.Collect()), kb())
	}
	_ = strings.TrimSpace
}
