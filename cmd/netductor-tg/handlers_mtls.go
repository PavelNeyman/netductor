package main

import (
	"strings"
)

func handleMtlsCB(token string, chat int64, msgID int, data string) {
	kb := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("mtls_list"), "m:mtls:list", "")},
		{btn(T("mtls_rotate_hint"), "m:mtls:rotate", "")},
		{btn(T("mtls_rollover_status"), "m:mtls:rollover", "")},
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
	if data == "m:mtls" || data == "m:mtls:" {
		reply(token, chat, msgID, T("mtls_title"), kb)
		return
	}
	if data == "m:mtls:list" {
		out := runND("mtls", "list")
		reply(token, chat, msgID, T("mtls_title")+"\n<pre>"+esc(out)+"</pre>", kb)
		return
	}
	if data == "m:mtls:rotate" {
		setState(chat, "wait_mtls_rotate", "")
		reply(token, chat, msgID, T("mtls_rotate_hint"), kb)
		return
	}
	if data == "m:mtls:rollover" {
		out := runND("mtls", "rollover", "status")
		reply(token, chat, msgID, T("mtls_title")+"\n<pre>"+esc(out)+"</pre>", kb)
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
