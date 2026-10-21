package main

import (
	"encoding/json"
	"strings"

	"github.com/PavelNeyman/netductor/internal/registry"
)

func handleRegistryCallback(token string, chat int64, msgID int, data string) bool {
	if data != "m:registry" && !strings.HasPrefix(data, "m:registry:") {
		return false
	}
	nav := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	if data == "m:registry" || data == "m:registry:status" {
		st := registry.StatusInfo()
		b, _ := json.MarshalIndent(st, "", "  ")
		nl := string([]byte{10})
		var body strings.Builder
		body.WriteString(T("reg_title") + nl)
		body.WriteString("<pre>" + esc(string(b)) + "</pre>" + nl)
		body.WriteString(`<tg-button-row align="left">`)
		body.WriteString(`<tg-button type="callback_data" style="primary" data="m:registry:ensure">` + T("reg_ensure") + `</tg-button>`)
		body.WriteString(`<tg-button type="callback_data" style="link" data="m:registry:crane">` + T("reg_crane") + `</tg-button>`)
		body.WriteString(`<tg-button type="callback_data" style="link" data="m:registry:catalog">` + T("reg_catalog") + `</tg-button>`)
		body.WriteString(`<tg-button type="callback_data" style="danger" data="m:registry:stop">` + T("reg_stop") + `</tg-button>`)
		body.WriteString(`</tg-button-row>` + nl)
		reply(token, chat, msgID, body.String(), nav)
		return true
	}
	back := map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:registry", "primary")}}}
	switch data {
	case "m:registry:ensure":
		st, err := registry.Ensure()
		msg := "✅ <b>" + T("reg_ensure") + "</b>\n<code>" + esc(st.Addr) + "</code>"
		if err != nil {
			msg = "❌ <b>" + T("reg_ensure") + "</b>\n<pre>" + esc(err.Error()) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:crane":
		path, err := registry.EnsureCrane()
		msg := "✅ <b>" + T("reg_crane") + "</b>\n<code>" + esc(path) + "</code>"
		if err != nil {
			msg = "❌ <b>" + T("reg_crane") + "</b>\n<pre>" + esc(err.Error()) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:stop":
		err := registry.Stop()
		msg := T("reg_stopped")
		if err != nil {
			msg = "❌ <b>" + T("reg_stop") + "</b>\n<pre>" + esc(err.Error()) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:catalog":
		list, err := registry.CatalogDetail()
		msg := T("reg_catalog") + "\n<i>" + T("reg_empty") + "</i>"
		if err != nil {
			msg = "❌ " + T("reg_catalog") + "\n<pre>" + esc(err.Error()) + "</pre>"
		} else if len(list) > 0 {
			var lines []string
			for _, r := range list {
				line := r.Name
				if len(r.Tags) > 0 {
					line += " [" + strings.Join(r.Tags, ", ") + "]"
				}
				lines = append(lines, "• <code>"+esc(line)+"</code>")
			}
			msg = T("reg_catalog") + "\n" + strings.Join(lines, "\n")
		}
		reply(token, chat, msgID, msg, back)
	default:
		return false
	}
	return true
}
