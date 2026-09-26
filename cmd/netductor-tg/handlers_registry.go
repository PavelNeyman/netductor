package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/registry"
)

func handleRegistryCallback(token string, chat int64, msgID int, data string) bool {
	if data != "m:registry" && !strings.HasPrefix(data, "m:registry:") {
		return false
	}
	nav := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("«", "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	if data == "m:registry" || data == "m:registry:status" {
		st := registry.StatusInfo()
		b, _ := json.MarshalIndent(st, "", "  ")
		nl := string([]byte{10})
		var body strings.Builder
		body.WriteString("📦 <b>Registry</b>" + nl)
		body.WriteString("<pre>" + esc(string(b)) + "</pre>" + nl)
		body.WriteString(`<tg-button-row align="left">`)
		body.WriteString(`<tg-button type="callback_data" style="primary" data="m:registry:ensure">Ensure</tg-button>`)
		body.WriteString(`<tg-button type="callback_data" style="link" data="m:registry:crane">Crane</tg-button>`)
		body.WriteString(`<tg-button type="callback_data" style="link" data="m:registry:catalog">📋</tg-button>`)
		body.WriteString(`<tg-button type="callback_data" style="danger" data="m:registry:stop">Stop</tg-button>`)
		body.WriteString(`</tg-button-row>` + nl)
		reply(token, chat, msgID, body.String(), nav)
		return true
	}
	back := map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:registry", "primary")}}}
	switch data {
	case "m:registry:ensure":
		st, err := registry.Ensure()
		msg := "✅ <b>Ensure</b>\n<code>" + esc(st.Addr) + "</code>"
		if err != nil {
			msg = "❌ <b>Ensure</b>\n<pre>" + esc(err.Error()) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:crane":
		path, err := registry.EnsureCrane()
		msg := "✅ <b>Crane</b>\n<code>" + esc(path) + "</code>"
		if err != nil {
			msg = "❌ <b>Crane</b>\n<pre>" + esc(err.Error()) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:stop":
		err := registry.Stop()
		msg := "✅ <b>Registry stopped</b>"
		if err != nil {
			msg = "❌ <b>Stop</b>\n<pre>" + esc(err.Error()) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:catalog":
		list, err := registry.CatalogDetail()
		msg := "📋 <b>Catalog</b>\n<i>(empty)</i>"
		if err != nil {
			msg = "❌ <b>Catalog</b>\n<pre>" + esc(err.Error()) + "</pre>"
		} else if len(list) > 0 {
			var lines []string
			for _, r := range list {
				line := r.Name
				if len(r.Tags) > 0 {
					line += " [" + strings.Join(r.Tags, ", ") + "]"
				}
				lines = append(lines, line)
			}
			msg = "📋 <b>Catalog</b>\n<pre>" + esc(strings.Join(lines, "\n")) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	default:
		reply(token, chat, msgID, fmt.Sprintf("unknown %s", data), nil)
	}
	return true
}
