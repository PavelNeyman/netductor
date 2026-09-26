package main

import (
	"strings"
)

func handleMtlsCB(token string, chat int64, msgID int, data string) {
	nav := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("« "+parentTools(), "m:tools", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}
	bodyActions := `<tg-button-row align="left">` +
		`<tg-button type="callback_data" style="link" data="m:mtls:list">` + T("mtls_list") + `</tg-button>` +
		`<tg-button type="callback_data" style="link" data="m:mtls:rotate">` + T("mtls_rotate_hint") + `</tg-button>` +
		`<tg-button type="callback_data" style="link" data="m:mtls:rollover">` + T("mtls_rollover_status") + `</tg-button>` +
		`</tg-button-row>`

	if data == "m:mtls" || data == "m:mtls:" {
		reply(token, chat, msgID, T("mtls_title")+"\n"+bodyActions, nav)
		return
	}
	if data == "m:mtls:list" {
		out := runND("mtls", "list")
		reply(token, chat, msgID, T("mtls_title")+"\n<pre>"+esc(out)+"</pre>\n"+bodyActions, nav)
		return
	}
	if data == "m:mtls:rotate" {
		setState(chat, "wait_mtls_rotate", "")
		reply(token, chat, msgID, T("mtls_rotate_hint")+"\n"+bodyActions, nav)
		return
	}
	if data == "m:mtls:rollover" {
		out := runND("mtls", "rollover", "status")
		reply(token, chat, msgID, T("mtls_title")+"\n<pre>"+esc(out)+"</pre>\n"+bodyActions, nav)
		return
	}
}

func tryMtlsMessage(token string, chat int64, text string) bool {
	if chatState[chat] != "wait_mtls_rotate" {
		return false
	}
	setState(chat, "", "")
	node := strings.TrimSpace(text)
	if node == "" {
		return true
	}
	out := runND("mtls", "rotate", node)
	sendHTML(token, chat, "<pre>"+esc(out)+"</pre>", map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("mtls"), "m:mtls", ""), btn(T("main_menu"), "m:menu", "primary")},
	}})
	return true
}
