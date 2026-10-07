package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/logs"
	"github.com/PavelNeyman/netductor/internal/notify"
)

func handleLogsCB(token string, chat int64, msgID int, data string) {
	switch data {
	case "m:logs":
		s := logs.FormatSchedule(logs.LoadSchedule())
		title := "📜 <b>Logs</b>"
		if getLang() != "en" {
			title = "📜 <b>Логи</b>"
		}
		body := title + "\n<pre>" + esc(s) + "</pre>\n" +
			"<i>Rotation vacuum journal; export last 1h for incidents</i>"
		if getLang() != "en" {
			body = title + "\n<pre>" + esc(s) + "</pre>\n" +
				"<i>Ротация journal; экспорт 1ч при инцидентах</i>"
		}
		reply(token, chat, msgID, body, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("📥 1h", "m:logs:export1h", "primary"), btn("🧹 Rotate", "m:logs:rotate", "")},
			{btn("04:00 MSK", "m:logs:sched:1:0", ""), btn("03:00 MSK", "m:logs:sched:0:0", "")},
			{btn(T("back"), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
		}})
	case "m:logs:rotate":
		out, err := logs.Rotate()
		msg := "<pre>" + esc(out) + "</pre>"
		if err != nil {
			msg = "⚠️ " + esc(err.Error()) + "\n" + msg
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(T("back"), "m:logs", "primary")},
		}})
	case "m:logs:export1h":
		path, err := logs.ExportHours(1)
		if err != nil {
			reply(token, chat, msgID, "⚠️ "+esc(err.Error()), map[string]any{"inline_keyboard": [][]map[string]any{
				{btn(T("back"), "m:logs", "primary")},
			}})
			return
		}
		_ = notify.SendDocument(path, "📎 logs last 1h")
		reply(token, chat, msgID, "✅ export → alerts chat\n<code>"+esc(path)+"</code>", map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(T("back"), "m:logs", "primary")},
		}})
	default:
		if strings.HasPrefix(data, "m:logs:sched:") {
			rest := strings.TrimPrefix(data, "m:logs:sched:")
			var h, m int
			fmt.Sscanf(rest, "%d:%d", &h, &m)
			s := logs.LoadSchedule()
			s.Hour, s.Minute, s.UTC = h, m, true
			if err := logs.SaveSchedule(s); err != nil {
				reply(token, chat, msgID, "⚠️ "+esc(err.Error()), map[string]any{"inline_keyboard": [][]map[string]any{
					{btn(T("back"), "m:logs", "primary")},
				}})
				return
			}
			reply(token, chat, msgID, "✅ "+esc(logs.FormatSchedule(s)), map[string]any{"inline_keyboard": [][]map[string]any{
				{btn(T("back"), "m:logs", "primary")},
			}})
			return
		}
		fmt.Fprintf(os.Stderr, "logs unknown cb %q\n", data)
	}
}
