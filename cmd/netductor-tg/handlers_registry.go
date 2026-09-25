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
		msg := "ok " + st.Addr
		if err != nil {
			msg = "err: " + err.Error()
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:crane":
		p, err := registry.EnsureCrane()
		msg := p
		if err != nil {
			msg = err.Error()
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:stop":
		err := registry.Stop()
		msg := "stopped"
		if err != nil {
			msg = err.Error()
		}
		reply(token, chat, msgID, msg, back)
	case "m:registry:catalog":
		list, err := registry.CatalogDetail()
		msg := "(empty)"
		if err != nil {
			msg = err.Error()
		} else if len(list) > 0 {
			var lines []string
			for _, r := range list {
				line := r.Name
				if len(r.Tags) > 0 {
					line += " [" + strings.Join(r.Tags, ", ") + "]"
				}
				lines = append(lines, line)
			}
			msg = "<pre>" + esc(strings.Join(lines, "\n")) + "</pre>"
		}
		reply(token, chat, msgID, msg, back)
	default:
		reply(token, chat, msgID, fmt.Sprintf("unknown %s", data), nil)
	}
	return true
}
