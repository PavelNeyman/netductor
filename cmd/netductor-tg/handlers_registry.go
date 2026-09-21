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
	if data == "m:registry" || data == "m:registry:status" {
		st := registry.StatusInfo()
		b, _ := json.MarshalIndent(st, "", "  ")
		body := "<pre>" + esc(string(b)) + "</pre>"
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("Ensure", "m:registry:ensure", ""), btn("Crane", "m:registry:crane", "")},
			{btn("Catalog", "m:registry:catalog", ""), btn("Stop", "m:registry:stop", "")},
			{btn("«", "m:menu", "")},
		}}
		reply(token, chat, msgID, body, kb)
		return true
	}
	switch data {
	case "m:registry:ensure":
		st, err := registry.Ensure()
		msg := "ok " + st.Addr
		if err != nil {
			msg = "err: " + err.Error()
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:registry", "")}}})
	case "m:registry:crane":
		p, err := registry.EnsureCrane()
		msg := p
		if err != nil {
			msg = err.Error()
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:registry", "")}}})
	case "m:registry:stop":
		err := registry.Stop()
		msg := "stopped"
		if err != nil {
			msg = err.Error()
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:registry", "")}}})
	case "m:registry:catalog":
		list, err := registry.Catalog()
		msg := "(empty)"
		if err != nil {
			msg = err.Error()
		} else if len(list) > 0 {
			msg = "<pre>" + esc(strings.Join(list, "\n")) + "</pre>"
		}
		reply(token, chat, msgID, msg, map[string]any{"inline_keyboard": [][]map[string]any{{btn("«", "m:registry", "")}}})
	default:
		reply(token, chat, msgID, fmt.Sprintf("unknown %s", data), nil)
	}
	return true
}
