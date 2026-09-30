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
		return map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("🔄", "m:stack", "primary"), btn("↩️ Rollback", "m:stack:rollback", ""), btn("🔧 WD", "m:stack:watchdog", "")},
			{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
		}}
	}
	switch {
	case data == "m:stack":
		st := stack.Collect()
		html := stack.FormatHTML(st)
		if ru {
			html += "\n\n<i>Rollback — предыдущие node+tg. WD — one-shot watchdog.</i>"
		} else {
			html += "\n\n<i>Rollback — prev node+tg. WD — one-shot watchdog.</i>"
		}
		reply(token, chat, msgID, html, kb())
	case data == "m:stack:rollback":
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
