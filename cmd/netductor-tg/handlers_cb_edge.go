package main

import "strings"

func handleCBEdgeApprove(token string, chat int64, msgID int, data string) bool {
	if strings.HasPrefix(data, "e:appr:") {
		id := strings.TrimPrefix(data, "e:appr:")
		out := runND("edge", "approve", id)
		out = maskTokenKV(out)
		reply(token, chat, msgID, "✅ <pre>"+esc(out)+"</pre>", routersKeyboard())
		return true
	}
	if strings.HasPrefix(data, "e:deny:") {
		id := strings.TrimPrefix(data, "e:deny:")
		out := runND("edge", "deny", id)
		reply(token, chat, msgID, "🚫 <pre>"+esc(out)+"</pre>", routersKeyboard())
		return true
	}
	if strings.HasPrefix(data, "e:rev:") {
		id := strings.TrimPrefix(data, "e:rev:")
		confirm := "Revoke <code>" + esc(id) + "</code>? Access token cleared; record kept."
		if getLang() != "en" {
			confirm = "Отозвать <code>" + esc(id) + "</code>? Token сброшен, запись останется."
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("✅ OK", "e:y_rev:"+id, "danger"), btn("❌", "m:edge", "")},
		}}
		reply(token, chat, msgID, confirm, kb)
		return true
	}
	if strings.HasPrefix(data, "e:y_rev:") {
		id := strings.TrimPrefix(data, "e:y_rev:")
		out := runND("edge", "revoke", id)
		reply(token, chat, msgID, "🔏 <pre>"+esc(out)+"</pre>", routersKeyboard())
		return true
	}
	if strings.HasPrefix(data, "e:del:") {
		id := strings.TrimPrefix(data, "e:del:")
		confirm := "DELETE <code>" + esc(id) + "</code>? Registry + mTLS material removed permanently."
		if getLang() != "en" {
			confirm = "УДАЛИТЬ <code>" + esc(id) + "</code>? Запись и mTLS material безвозвратно."
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("✅ DELETE", "e:y_del:"+id, "danger"), btn("❌", "m:edge", "")},
		}}
		reply(token, chat, msgID, confirm, kb)
		return true
	}
	if strings.HasPrefix(data, "e:y_del:") {
		id := strings.TrimPrefix(data, "e:y_del:")
		out := runND("edge", "delete", id)
		reply(token, chat, msgID, "🗑 <pre>"+esc(out)+"</pre>", routersKeyboard())
		return true
	}
	return false
}
