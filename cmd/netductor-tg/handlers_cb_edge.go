package main

import "strings"

func handleCBEdgeApprove(token string, chat int64, msgID int, data string) bool {
	if strings.HasPrefix(data, "e:appr:") {
		id := strings.TrimPrefix(data, "e:appr:")
		out := runND("edge", "approve", id)
		// mask token=... in output
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
	return false
}
